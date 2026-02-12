import type { Configuration } from 'webpack';
import { merge } from 'webpack-merge';
import CopyWebpackPlugin from 'copy-webpack-plugin';
import path from 'path';
import grafanaConfig, { type Env } from './.config/webpack/webpack.config';

/**
 * Extended webpack configuration for the JAOPS Yamcs plugin.
 * This file extends the default Grafana plugin webpack config.
 */
const config = async (env: Env): Promise<Configuration> => {
  const baseConfig = await grafanaConfig(env);

  return merge(baseConfig, {
    plugins: [
      // Copy screenshots for plugin.json marketplace listing
      new CopyWebpackPlugin({
        patterns: [
          {
            from: path.resolve(process.cwd(), 'screenshots'),
            to: 'screenshots',
            noErrorOnMissing: true,
          },
        ],
      }),
    ],
  });
};

export default config;
