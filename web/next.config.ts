import type { NextConfig } from "next";
import { PHASE_DEVELOPMENT_SERVER } from "next/constants";
import fs from 'node:fs';
import path from 'node:path';

function resolveAppVersion() {
  const envVersion = process.env.NEXT_PUBLIC_APP_VERSION?.trim();
  if (envVersion) {
    return envVersion;
  }

  try {
    const versionFile = path.resolve(process.cwd(), '..', 'internal', 'conf', 'version.go');
    const content = fs.readFileSync(versionFile, 'utf8');
    const match = content.match(/Version\s*=\s*"([^"]+)"/);
    if (match?.[1]) {
      return `v${match[1].replace(/^v/i, '')}`;
    }
  } catch {
    // ignore and fall back to unknown
  }

  return 'unknown';
}

const createNextConfig = (phase: string): NextConfig => ({
  reactCompiler: true,
  output: "export",
  images: {
    unoptimized: true,
  },
  optimizePackageImports: [
    "recharts",
    "@lobehub/icons",
    "lucide-react",
    "@radix-ui/react-icons",
  ],
  env: {
    NEXT_PUBLIC_APP_VERSION: resolveAppVersion(),
  },
  ...(phase === PHASE_DEVELOPMENT_SERVER ? {} : { assetPrefix: "./" }),
});

export default createNextConfig;

