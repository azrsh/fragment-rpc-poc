import { createServer } from "node:http";
import { readFile } from "node:fs/promises";
import { resolve, extname } from "node:path";
import { connectNodeAdapter } from "@connectrpc/connect-node";
import { registerOperations } from "../generated/routes.js";
import { startBackends } from "./backends.js";
import { createResolvers } from "./resolvers.js";
import { serveGrpc } from "./network.js";

const backends = await startBackends((service, id) => console.log(`gRPC ${service}(${JSON.stringify(id)})`));
const routes = (router: Parameters<typeof registerOperations>[0]) => registerOperations(router, createResolvers(backends.user.url, backends.organization.url));
let grpc: Awaited<ReturnType<typeof serveGrpc>>;
try { grpc = await serveGrpc(routes, Number(process.env.GRPC_PORT ?? 50051)); }
catch (error) { await backends.close(); throw error; }
const adapter = connectNodeAdapter({ routes });
const root = resolve("dist");
const mime: Record<string, string> = { ".html": "text/html; charset=utf-8", ".js": "text/javascript", ".css": "text/css", ".svg": "image/svg+xml" };
const web = createServer(async (req, res) => {
  if (req.url?.startsWith("/app.v1.AppService/")) { adapter(req, res); return; }
  if (req.method !== "GET" && req.method !== "HEAD") { res.writeHead(405).end(); return; }
  try {
    const url = new URL(req.url ?? "/", "http://localhost");
    const path = resolve(root, "." + decodeURIComponent(url.pathname === "/" ? "/index.html" : url.pathname));
    if (!path.startsWith(root + "/")) { res.writeHead(403).end(); return; }
    const data = await readFile(path);
    res.writeHead(200, { "Content-Type": mime[extname(path)] ?? "application/octet-stream" });
    res.end(req.method === "HEAD" ? undefined : data);
  } catch { res.writeHead(404).end("Run npm run build before opening the demo."); }
});
try {
  await new Promise<void>((resolve, reject) => {
    web.once("error", reject);
    web.listen(Number(process.env.PORT ?? 3000), "127.0.0.1", resolve);
  });
} catch (error) { await Promise.all([grpc.close(), backends.close()]); throw error; }
console.log(`Demo: http://127.0.0.1:${process.env.PORT ?? 3000}`);
console.log(`Generated gRPC API: ${grpc.url}/app.v1.AppService`);
console.log(`Backends: users ${backends.user.url}, organizations ${backends.organization.url}`);
let closing = false;
async function close() {
  if (closing) return;
  closing = true;
  web.closeAllConnections();
  await Promise.all([new Promise<void>(resolve => web.close(() => resolve())), grpc.close(), backends.close()]);
}
process.once("SIGINT", () => void close());
process.once("SIGTERM", () => void close());
