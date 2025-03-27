import useWebSocket from "react-use-websocket";
import { useEffect, useRef, useState } from "react";
import { CodecInfoType, GameInfoType } from "@/lib/types";
import { Button } from "@/components/ui/button";
import useGamepad from "@/hooks/use-gamepad";
import { toGamepadStateDto } from "@/lib/utils";

type StatsType = {
  timestamp: number;
  bytesReceived: number;
  bitrate: number;
  nackCount: number;
  packetsReceived: number;
  frameRate: number;
  resolution: string;
  totalInterFrameDelay: number;
  interFrameDelay: number;
  rtt: number | undefined;
  codec: string;
  loss: number;
};

export default function Gameplay(props: {
  server: string;
  game: GameInfoType;
  codec: CodecInfoType;
  record: boolean;
  onExit?: () => void;
}) {
  const [showTopBar, setShowTopBar] = useState(true);
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

  // generate ws://xxx from http(s):// url
  const wsUrl = props.server.replace(/^http/, "ws");
  const ws = useWebSocket(`${wsUrl}/webrtc`, {
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
      ws.sendMessage(
        JSON.stringify({
          game_config: props.game,
          codec_config: props.codec,
        }),
      );
    },
  });
  const [stats, setStats] = useState<StatsType>({
    timestamp: 0,
    bytesReceived: 0,
    bitrate: 0, // Mbps
    nackCount: 0,
    packetsReceived: 0,
    frameRate: 0,
    resolution: "0x0",
    totalInterFrameDelay: 0, // s
    interFrameDelay: 0,
    rtt: undefined,
    codec: "",
    loss: 0,
  });

  // Stats collection interval
  useEffect(() => {
    const statsInterval = setInterval(async () => {
      if (!peerConnectionRef.current) return;

      const stats = await peerConnectionRef.current.getStats();
      for (const [_, stat] of stats.entries()) {
        if (stat.type === "codec" && stat.mimeType.startsWith("video")) {
          setStats((prev) => ({
            ...prev,
            codec: stat.mimeType,
          }));
        }
        if (stat.type === "candidate-pair" && stat.state === "succeeded") {
          setStats((prev) => ({
            ...prev,
            rtt: stat.currentRoundTripTime,
          }));
        }
        if (stat.type === "inbound-rtp" && stat.kind === "video") {
          setStats((prev) => ({
            ...prev,
            timestamp: stat.timestamp,
            bytesReceived: stat.bytesReceived,
            bitrate:
              ((stat.bytesReceived - prev.bytesReceived) * 8) /
              1_000_000 /
              ((stat.timestamp - prev.timestamp) / 1000),
            nackCount: stat.nackCount,
            packetsReceived: stat.packetsReceived,
            frameRate: stat.framesPerSecond,
            resolution:
              stat.frameWidth && stat.frameHeight
                ? `${stat.frameWidth}x${stat.frameHeight}`
                : prev.resolution,
            totalInterFrameDelay: stat.totalInterFrameDelay,
            interFrameDelay:
              (stat.totalInterFrameDelay - prev.totalInterFrameDelay) /
              ((stat.timestamp - prev.timestamp) / 1000),
            loss:
              ((stat.nackCount - prev.nackCount) /
                (stat.packetsReceived - prev.packetsReceived)) *
              100,
          }));
        }
      }
    }, 1000);

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

        if (props.record) {
          const mimeType = "video/mp4; codecs=avc1.42E01E";
          const recordedChunks: Blob[] = [];
          const mediaRecorder = new MediaRecorder(stream, { mimeType });
          mediaRecorder.ondataavailable = (event) => {
            if (event.data && event.data.size > 0) {
              recordedChunks.push(event.data);
            }
          };
          mediaRecorder.onstop = () => {
            // Combine recorded chunks into a Blob
            const videoBlob = new Blob(recordedChunks, { type: mimeType });
            // Create a URL for the Blob
            const videoUrl = URL.createObjectURL(videoBlob);
            // Create a download link
            const gameName = props.game.game_display_name || "unknown";
            // Create safe filename (remove special characters)
            const safeGameName = gameName.replace(/[^\w\s-]/g, '').replace(/\s+/g, '-');
            // 时间戳，东八区
            const date = new Intl.DateTimeFormat('zh-CN', { //年月日
              timeZone: 'Asia/Shanghai',
              year: 'numeric',
              month: '2-digit',
              day: '2-digit'
            }).format(new Date()).replace(/\//g, '-');

            const time = new Intl.DateTimeFormat('zh-CN', { //时分秒
              timeZone: 'Asia/Shanghai',
              hour: '2-digit',
              minute: '2-digit',
              second: '2-digit',
              hour12: false
            }).format(new Date()).replace(/:/g, '-');

            const timestamp = `${date}_${time}`;
            // Create a download link
            const downloadLink = document.createElement("a");
            downloadLink.href = videoUrl;
            downloadLink.download = `${safeGameName}_${timestamp}.mp4`;
            downloadLink.click();
          };
          mediaRecorder.start(5000);
        }

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
    <div className="max-h-svh">
      {/* A fullscreen transparent div on top that captures onClick event */}
      <div
        className="absolute inset-0 z-40"
        onClick={() => setShowTopBar((prev) => !prev)}
      />

      {/* Video takes up full screen */}
      <video
        ref={videoRef}
        autoPlay
        muted
        playsInline
        className="absolute inset-0 mx-auto mb-0 mt-auto h-full max-h-svh w-full touch-none object-contain"
      />

      {/* Floating top bar */}
      {showTopBar && (
        <div className="absolute left-0 right-0 top-0 z-50 flex touch-none items-center justify-between bg-black/50 px-4 backdrop-blur-sm">
          <h1 className="hidden text-lg font-bold text-white sm:block">
            PionGS Gameplay
          </h1>
          <div className="flex flex-wrap space-x-4 text-sm text-white/80">
            <div className="flex flex-col">
              <span className="text-xs text-white/60">Bitrate</span>
              <span>{stats.bitrate.toFixed(2)} Mbps</span>
            </div>
            <div className="flex flex-col">
              <span className="text-xs text-white/60">RTT</span>
              <span>
                {stats.rtt ? (stats.rtt * 1000).toFixed(2) + " ms" : "N/A"}
              </span>
            </div>
            <div className="flex flex-col">
              <span className="text-xs text-white/60">Loss</span>
              <span>{stats.loss.toFixed(2) + "%"}</span>
            </div>
            <div className="flex flex-col">
              <span className="text-xs text-white/60">Frame Rate</span>
              <span>{stats.frameRate} fps</span>
            </div>
            <div className="flex flex-col">
              <span className="text-xs text-white/60">Resolution</span>
              <span>{stats.resolution}</span>
            </div>
            <div className="flex flex-col">
              <span className="text-xs text-white/60">Codec</span>
              <span>{stats.codec}</span>
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
      )}
    </div>
  );
}
