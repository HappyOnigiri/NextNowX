import react from "@vitejs/plugin-react";
import { randomUUID } from "node:crypto";
import { readFileSync } from "node:fs";
import { resolve } from "node:path";
import { createViteLicensePlugin } from "rollup-license-plugin";
import type { Plugin } from "vite";
import { defineConfig } from "vitest/config";

const apiOrigin = "http://127.0.0.1:7332";
const licenseReportPath = resolve(
  __dirname,
  "../internal/webui/dist/oss-licenses.json",
);
const packageManifest: unknown = JSON.parse(
  readFileSync(resolve(__dirname, "../package.json"), "utf8"),
);
if (
  typeof packageManifest !== "object" ||
  packageManifest === null ||
  !("version" in packageManifest) ||
  typeof packageManifest.version !== "string"
) {
  throw new Error("root package.json has no version");
}
const developmentVersion = `${packageManifest.version}-dev`;
const devOrigins = new Set([
  "http://127.0.0.1:7331",
  "http://localhost:7331",
  "http://[::1]:7331",
]);
const demoPlaceholder = /__PRX_DEMO__/g;
const demoSessionPlaceholder = /__PRX_DEMO_SESSION__/g;
const demoMode = process.env["PRX_DEMO"] === "true";
// 開発サーバのプロセスを表す ID である。閉じた警告は読み込み直しても戻らないが、
// 開発サーバを起動し直すと ID が変わって戻る。
const demoSession = randomUUID();

function serveLicenseReport(): Plugin {
  return {
    name: "serve-license-report",
    apply: "serve",
    configureServer(server) {
      server.middlewares.use(
        "/oss-licenses.json",
        (_request, response, next) => {
          try {
            response.setHeader("Content-Type", "application/json");
            response.end(readFileSync(licenseReportPath));
          } catch (error) {
            next(error);
          }
        },
      );
    },
  };
}

// 本番の index.html は Go のハンドラがプレースホルダを埋める。開発サーバは Vite が配信するので、
// demo の API を相手にしているときは同じメタデータをここで注入する。
function injectDemoMode(): Plugin {
  return {
    name: "inject-demo-mode",
    apply: "serve",
    transformIndexHtml(html) {
      return html
        .replace(demoPlaceholder, String(demoMode))
        .replace(demoSessionPlaceholder, demoSession);
    },
  };
}

export default defineConfig({
  plugins: [
    react(),
    createViteLicensePlugin(),
    serveLicenseReport(),
    injectDemoMode(),
  ],
  define: {
    "import.meta.env.APP_VERSION": JSON.stringify(developmentVersion),
  },
  server: {
    port: 7331,
    strictPort: true,
    proxy: {
      "/prx.v1": {
        target: apiOrigin,
        changeOrigin: true,
        configure(proxy) {
          proxy.on("proxyReq", (proxyRequest, request) => {
            const origin = request.headers.origin;
            // 既知のローカル Vite 開発オリジン以外の呼び出し元には、
            // API の origin チェックをそのまま効かせる。
            if (origin && devOrigins.has(origin))
              proxyRequest.setHeader("origin", apiOrigin);
          });
        },
      },
    },
  },
  build: {
    outDir: resolve(__dirname, "../internal/webui/dist"),
    emptyOutDir: false,
    // 成果物は 127.0.0.1 のローカル配信なので転送量を気にする必要がなく、
    // 分割せず単一チャンクのままにしている。既定の 500 kB では警告が出るため引き上げる。
    chunkSizeWarningLimit: 2000,
  },
  test: {
    environment: "jsdom",
    setupFiles: "./tests/setup.ts",
    exclude: ["tests/e2e/**", "node_modules/**"],
    // `make ci` は複数のターゲットを CPU 数だけ並列に走らせるため、CPU の奪い合いで
    // 個々のテストの実時間が単独実行の 15〜19 倍まで伸びる。既定値（5000 / 10000）では
    // 打ち切られるので引き上げる。テスト自体が重いわけではない。
    testTimeout: 15_000,
    hookTimeout: 15_000,
    coverage: {
      provider: "v8",
      reporter: ["text", "json-summary", "json"],
      include: ["src/**/*.{ts,tsx}"],
      exclude: ["src/gen/**", "src/**/*.d.ts"],
      thresholds: {
        statements: 43.25,
        branches: 38.37,
        lines: 43.77,
      },
    },
  },
});
