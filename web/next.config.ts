import type { NextConfig } from "next";
import path from "node:path";

const nextConfig: NextConfig = {
  turbopack: {
    root: path.join(__dirname),
  },

  // Tree-shake heavy libs (framer-motion, recharts) so dev process
  // doesn't load every module. lucide-react is auto-optimised.
  experimental: {
    optimizePackageImports: ["framer-motion", "recharts"],
  },

  // Drop unused pages from the dev buffer aggressively — RAM stops
  // growing as you click around the sidebar.
  onDemandEntries: {
    maxInactiveAge: 25_000,
    pagesBufferLength: 2,
  },

  productionBrowserSourceMaps: false,
  poweredByHeader: false,
};

export default nextConfig;
