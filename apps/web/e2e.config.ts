import type { E2EConfig } from "e2e";
import { web } from "@e2e-dev/web";

export default {
  targets: [
    {
      engine: web(),
      app: {
        url: "http://localhost:3100",
        command: {
          executable: "pnpm",
          args: ["dev:test"],
          log: ".e2e/logs/app.log",
        },
      },
    },
  ],
} satisfies E2EConfig;
