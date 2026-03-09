import type { NextConfig } from "next";
import { PHASE_DEVELOPMENT_SERVER } from "next/constants";

const createNextConfig = (phase: string): NextConfig => ({
  reactCompiler: true,
  output: "export",
  ...(phase === PHASE_DEVELOPMENT_SERVER ? {} : { assetPrefix: "./" }),
});

export default createNextConfig;

