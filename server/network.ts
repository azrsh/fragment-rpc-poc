import { createServer, type Http2Server, type ServerHttp2Session } from "node:http2";
import { connectNodeAdapter } from "@connectrpc/connect-node";
import type { ConnectRouter } from "@connectrpc/connect";

export async function serveGrpc(routes: (router: ConnectRouter) => void, port = 0) {
  const server: Http2Server = createServer(connectNodeAdapter({ routes }));
  const sessions = new Set<ServerHttp2Session>();
  server.on("session", session => {
    sessions.add(session);
    session.on("close", () => sessions.delete(session));
  });
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(port, "127.0.0.1", resolve);
  });
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("Missing server address");
  return {
    url: `http://127.0.0.1:${address.port}`,
    close: () => new Promise<void>((resolve, reject) => {
      for (const session of sessions) session.destroy();
      server.close(error => error ? reject(error) : resolve());
    }),
  };
}
