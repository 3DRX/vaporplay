import {
  Table,
  TableBody,
  TableCaption,
  TableHead,
  TableHeader,
  TableRow,
} from "./ui/table";

export default function CodecCapabilities() {
  const receiverVideoCapabilities = RTCRtpReceiver.getCapabilities("video");
  const receiverAudioCapabilities = RTCRtpReceiver.getCapabilities("audio");

  return (
    <div className="p-4">
      <h2 className="mb-4 text-xl font-bold">Codec Capabilities</h2>
      <h3 className="text-lg">Video</h3>
      <div className="flex flex-row gap-5">
        <Table>
          <TableCaption>codecs</TableCaption>
          <TableHeader>
            <TableRow>
              <TableHead>clockRate</TableHead>
              <TableHead>mimeType</TableHead>
              <TableHead>sdpFmtpLine</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {receiverVideoCapabilities &&
              receiverVideoCapabilities.codecs.map((codec, index) => (
                <TableRow key={index}>
                  <td>{codec.clockRate}</td>
                  <td>{codec.mimeType}</td>
                  <td>{codec.sdpFmtpLine}</td>
                </TableRow>
              ))}
          </TableBody>
        </Table>
        <Table>
          <TableCaption>headerExtensions</TableCaption>
          <TableHeader>
            <TableRow>
              <TableHead>direction</TableHead>
              <TableHead>uri</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {receiverVideoCapabilities &&
              receiverVideoCapabilities.headerExtensions.map(
                (headerExtension, index) => (
                  <TableRow key={index}>
                    {/*@ts-ignore*/}
                    <td>{headerExtension.direction}</td>
                    <td>{headerExtension.uri}</td>
                  </TableRow>
                ),
              )}
          </TableBody>
        </Table>
      </div>
      <h3 className="text-lg">Audio</h3>
      <div className="flex flex-row gap-5">
        <Table>
          <TableCaption>codecs</TableCaption>
          <TableHeader>
            <TableRow>
              <TableHead>clockRate</TableHead>
              <TableHead>mimeType</TableHead>
              <TableHead>sdpFmtpLine</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {receiverAudioCapabilities &&
              receiverAudioCapabilities.codecs.map((codec, index) => (
                <TableRow key={index}>
                  <td>{codec.clockRate}</td>
                  <td>{codec.mimeType}</td>
                  <td>{codec.sdpFmtpLine}</td>
                </TableRow>
              ))}
          </TableBody>
        </Table>
        <Table>
          <TableCaption>headerExtensions</TableCaption>
          <TableHeader>
            <TableRow>
              <TableHead>direction</TableHead>
              <TableHead>uri</TableHead>
            </TableRow>
          </TableHeader>
          <TableBody>
            {receiverAudioCapabilities &&
              receiverAudioCapabilities.headerExtensions.map(
                (headerExtension, index) => (
                  <TableRow key={index}>
                    {/*@ts-ignore*/}
                    <td>{headerExtension.direction}</td>
                    <td>{headerExtension.uri}</td>
                  </TableRow>
                ),
              )}
          </TableBody>
        </Table>
      </div>
    </div>
  );
}
