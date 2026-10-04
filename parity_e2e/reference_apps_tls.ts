import { execFileSync } from "node:child_process";
import { readFileSync } from "node:fs";
import { request as httpRequest } from "node:http";
import { createServer } from "node:https";
import type { Socket } from "node:net";
import path from "node:path";

/** Serve the fixture over HTTPS so every engine can use Secure session cookies. */
export async function startLoopbackTLSProxy(upstreamURL: string, tempDir: string) {
  const upstream = new URL(upstreamURL);
  if (upstream.protocol !== "http:" || upstream.hostname !== "127.0.0.1") {
    throw new Error("reference-app TLS proxy requires a loopback HTTP upstream");
  }
  const keyPath = path.join(tempDir, "fixture-key.pem");
  const certPath = path.join(tempDir, "fixture-cert.pem");
  execFileSync("openssl", [
    "req", "-x509", "-newkey", "rsa:2048", "-nodes", "-days", "1",
    "-subj", "/CN=127.0.0.1", "-addext", "subjectAltName=IP:127.0.0.1",
    "-keyout", keyPath, "-out", certPath,
  ], { stdio: "ignore" });

  const sockets = new Set<Socket>();
  let baseURL = "";
  const forward = (req: import("node:http").IncomingMessage) => {
    const headers = { ...req.headers, "x-forwarded-proto": "https" };
    return httpRequest({
      hostname: upstream.hostname,
      port: upstream.port,
      path: req.url,
      method: req.method,
      headers,
    });
  };
  const server = createServer({ key: readFileSync(keyPath), cert: readFileSync(certPath) }, (req, res) => {
    const outgoing = forward(req);
    outgoing.on("response", (incoming) => {
      res.writeHead(incoming.statusCode ?? 502, incoming.headers);
      incoming.pipe(res);
    });
    outgoing.on("error", () => {
      if (!res.headersSent) res.writeHead(502);
      res.end();
    });
    req.on("aborted", () => outgoing.destroy());
    req.pipe(outgoing);
  });
  server.on("connection", (socket) => {
    sockets.add(socket);
    socket.once("close", () => sockets.delete(socket));
  });
  server.on("upgrade", (req, socket, head) => {
    const outgoing = forward(req);
    outgoing.on("upgrade", (response, backend, backendHead) => {
      socket.write(`HTTP/1.1 ${response.statusCode} ${response.statusMessage}\r\n`);
      for (let index = 0; index < response.rawHeaders.length; index += 2) {
        socket.write(`${response.rawHeaders[index]}: ${response.rawHeaders[index + 1]}\r\n`);
      }
      socket.write("\r\n");
      if (backendHead.length) socket.write(backendHead);
      if (head.length) backend.write(head);
      socket.once("close", () => backend.destroy());
      backend.once("close", () => socket.destroy());
      socket.on("error", () => backend.destroy());
      backend.on("error", () => socket.destroy());
      socket.pipe(backend).pipe(socket);
    });
    outgoing.on("response", () => socket.destroy());
    outgoing.on("error", () => socket.destroy());
    outgoing.end();
  });
  await new Promise<void>((resolve, reject) => {
    server.once("error", reject);
    server.listen(0, "127.0.0.1", resolve);
  });
  const address = server.address();
  if (!address || typeof address === "string") throw new Error("TLS proxy did not bind a loopback port");
  baseURL = `https://127.0.0.1:${address.port}`;
  return {
    baseURL,
    stop: () => new Promise<void>((resolve, reject) => {
      server.close((error) => error ? reject(error) : resolve());
      for (const socket of sockets) socket.destroy();
    }),
  };
}
