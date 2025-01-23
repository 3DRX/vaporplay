import useWebSocket from "react-use-websocket";
import { Button } from "./ui/button";
import { useRef } from "react";
import { GameInfoType } from "@/lib/types";

export default function Gameplay(props: {
  server: string;
  game: GameInfoType;
  onExit?: () => void;
}) {
  const peerConnectionRef = useRef<RTCPeerConnection | null>(null); // Store RTCPeerConnection reference
  const videoRef = useRef<HTMLVideoElement | null>(null);
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

  const handleSDPOffer = async (offer: RTCSessionDescriptionInit) => {
    // Create a new RTCPeerConnection
    const pc = new RTCPeerConnection({
      iceServers: [{ urls: "stun:stun.l.google.com:19302" }],
    });

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
    <div>
      <h1>Gameplay</h1>
      <Button
        variant="link"
        onClick={() => {
          peerConnectionRef.current?.close();
          props.onExit && props.onExit();
        }}
      >
        Exit
      </Button>
      <video ref={videoRef} autoPlay muted className="max-h-[90vh] w-full" />
    </div>
  );
}
