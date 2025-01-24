import useWebSocket from "react-use-websocket";
import { useEffect, useRef, useState } from "react";
import { GameInfoType } from "@/lib/types";
import { Button } from "@/components/ui/button";
import useGamepad from "@/hooks/use-gamepad";
import { toGamepadStateDto } from "@/lib/utils";

export default function Gameplay(props: {
  server: string;
  game: GameInfoType;
  onExit?: () => void;
}) {
  const peerConnectionRef = useRef<RTCPeerConnection | null>(null);
  const videoRef = useRef<HTMLVideoElement | null>(null);
  const dataChannelRef = useRef<RTCDataChannel | null>(null);

  useGamepad({
    onGamepadStateChange: (gamepadState, _) => {
      if (
        dataChannelRef.current &&
        dataChannelRef.current.label === "controller"
      ) {
        dataChannelRef.current.send(
          JSON.stringify(toGamepadStateDto(gamepadState)),
        );
      }
    },
  });

  const ws = useWebSocket(`${props.server}/webrtc`, {
    onMessage: (message) => {
      const signal = JSON.parse(message.data);
      if (signal.sdp) {
        console.log("Received SDP offer", signal);
        handleSDPOffer(signal);
      } else if (signal.candidate) {
        console.log("Received ICE candidate", signal);
        handleICECandidate(signal);
      }
    },
    onOpen: () => {
      ws.sendMessage(JSON.stringify(props.game));
    },
  });
  const [stats, setStats] = useState({
    timestamp: 0,
    bytesReceived: 0,
    bitrate: 0, // Mbps
    packetsLost: 0,
    frameRate: 0,
    resolution: "0x0",
    totalInterFrameDelay: 0, // s
    interFrameDelay: 0,
  });

  // Stats collection interval
  useEffect(() => {
    const statsInterval = setInterval(async () => {
      if (!peerConnectionRef.current) return;

      const stats = await peerConnectionRef.current.getStats();
      for (const [_, stat] of stats.entries()) {
        // @ts-ignore
        if (stat.type === "inbound-rtp" && stat.kind === "video") {
          console.log(stat);
          setStats((prev) => ({
            timestamp: stat.timestamp,
            bytesReceived: stat.bytesReceived,
            bitrate:
              ((stat.bytesReceived - prev.bytesReceived) * 8) /
              1_000_000 /
              ((stat.timestamp - prev.timestamp) / 1000),
            packetsLost: stat.packetsLost,
            frameRate: stat.framesPerSecond,
            resolution: `${stat.frameWidth}x${stat.frameHeight}`,
            totalInterFrameDelay: stat.totalInterFrameDelay,
            interFrameDelay:
              (stat.totalInterFrameDelay - prev.totalInterFrameDelay) /
              ((stat.timestamp - prev.timestamp) / 1000),
          }));
        }
      }
    }, 3000);

    return () => clearInterval(statsInterval);
  }, []);

  const handleSDPOffer = async (offer: RTCSessionDescriptionInit) => {
    const pc = new RTCPeerConnection({
      iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
    });

    pc.ondatachannel = (event) => {
      if (event.channel) {
        console.log("Data channel is created!");
        dataChannelRef.current = event.channel;
      }
    };

    pc.oniceconnectionstatechange = () => {
      console.log("ICE connection state:", pc.iceConnectionState);
    };

    pc.ontrack = (event) => {
      console.log("on track");
      // Check if the track is a video track
      if (event.track.kind === "video" && videoRef.current) {
        // Create a new media stream and add the video track to it
        const stream = new MediaStream();
        stream.addTrack(event.track);

        // Bind the stream to the video element
        videoRef.current.srcObject = stream;
        console.log("Video track added to video element");
      }
    };

    peerConnectionRef.current = pc;

    // Set remote SDP offer
    await pc.setRemoteDescription(new RTCSessionDescription(offer));

    // Create SDP answer
    const answer = await pc.createAnswer();
    await pc.setLocalDescription(answer);

    // Send SDP answer to the server
    sendSDPAnswer(answer);
  };

  const sendSDPAnswer = (answer: RTCSessionDescriptionInit) => {
    const message = JSON.stringify(answer);
    ws.sendMessage(message);
    console.log("Sent SDP answer:", answer);
  };

  const handleICECandidate = (candidate: RTCIceCandidateInit) => {
    if (peerConnectionRef.current) {
      peerConnectionRef.current
        .addIceCandidate(new RTCIceCandidate(candidate))
        .then(() => {
          console.log("ICE candidate added successfully");
        })
        .catch((error) => {
          console.error("Error adding ICE candidate:", error);
        });
    }
  };

  return (
    <div className="relative h-screen w-screen">
      {/* Video takes up full screen */}
      <video
        ref={videoRef}
        autoPlay
        muted
        className="absolute inset-0 mb-0 mt-auto h-auto w-full object-cover"
      />

      {/* Floating top bar */}
      <div className="absolute left-0 right-0 top-0 flex items-center justify-between bg-black/50 px-4 backdrop-blur-sm">
        <h1 className="text-lg font-bold text-white">PionGS Gameplay</h1>
        <div className="flex space-x-4 text-sm text-white/80">
          <div className="flex flex-col">
            <span className="text-xs text-white/60">Bitrate</span>
            <span>{stats.bitrate.toFixed(2)} Mbps</span>
          </div>
          <div className="flex flex-col">
            <span className="text-xs text-white/60">Frame Rate</span>
            <span>{stats.frameRate} fps</span>
          </div>
          <div className="flex flex-col">
            <span className="text-xs text-white/60">Resolution</span>
            <span>{stats.resolution}</span>
          </div>
        </div>
        <Button
          variant="link"
          onClick={() => {
            peerConnectionRef.current?.close();
            props.onExit && props.onExit();
          }}
          className="h-5 text-white transition-colors hover:text-gray-300"
        >
          Exit
        </Button>
      </div>
    </div>
  );
}
