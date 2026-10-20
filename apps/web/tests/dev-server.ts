import { spawn } from "node:child_process";
import { createMockServer } from "./mock-backend.ts";

const MOCK_PORT = 8088;
const NEXT_PORT = 3100;

const mockServer = createMockServer();

mockServer.listen(MOCK_PORT, "127.0.0.1", () => {
  console.log(`[mock-backend] Listening on http://127.0.0.1:${MOCK_PORT}`);

  const nextProcess = spawn("pnpm", ["dev", "-p", String(NEXT_PORT)], {
    env: {
      ...process.env,
      PORT: String(NEXT_PORT),
      CROSSWATC_API_URL: `http://127.0.0.1:${MOCK_PORT}`,
      CROSSWATC_DEV_USER: "admin",
      CROSSWATC_DEV_SQUADS: "sre,platform,payments",
      CROSSWATC_DEV_SRE: "true",
    },
    stdio: "inherit",
  });

  const cleanup = () => {
    mockServer.close();
    nextProcess.kill("SIGTERM");
    process.exit(0);
  };

  process.on("SIGINT", cleanup);
  process.on("SIGTERM", cleanup);

  nextProcess.on("exit", (code) => {
    mockServer.close();
    process.exit(code ?? 0);
  });
});
