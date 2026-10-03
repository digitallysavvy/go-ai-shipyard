import path from 'node:path';
import type { NextConfig } from 'next';

const nextConfig: NextConfig = {
  reactStrictMode: true,
  // Pin the workspace root to this app so a stray lockfile higher up the
  // directory tree isn't picked up.
  turbopack: { root: path.resolve(import.meta.dirname) },
  agentRules: false,
};

export default nextConfig;
