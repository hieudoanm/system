import { defineConfig, globalIgnores } from 'eslint/config';

const eslintConfig = defineConfig([
  // Override default ignores of eslint-config-next.
  globalIgnores([
    // Default ignores of eslint-config-next:
    '.next/**',
    'build/**',
    'out/**',
    'next-env.d.ts',
    'jest.config.ts',
    'jest.setup.ts',
    'src-tauri/target/**',
  ]),
]);

export default eslintConfig;
