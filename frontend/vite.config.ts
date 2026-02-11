import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig(({ mode }) => {
  // DEBUG 模式：不压缩、不混淆、生成 sourcemap
  const isDebug = process.env.DEBUG === 'true' || mode === 'debug'

  return {
    base: '/www/',
    plugins: [vue()],
    resolve: {
      alias: {
        '@': resolve(__dirname, 'src'),
      },
    },
    build: {
      outDir: resolve(__dirname, '../embed/www'),
      emptyOutDir: true,
      // DEBUG 模式：不压缩代码
      minify: isDebug ? false : 'esbuild',
      // DEBUG 模式：生成 sourcemap
      sourcemap: isDebug ? 'inline' : false,
      rollupOptions: {
        output: {
          manualChunks: (id) => {
            if (id.includes('node_modules')) {
              if (id.includes('ant-design-vue')) {
                return 'vendor-antd'
              }
              return 'vendor'
            }
          },
        },
      },
      chunkSizeWarningLimit: 1000,
    },
    server: {
      port: 5173,
      proxy: {
        '/api': {
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
        '/image': {
          target: 'http://localhost:8080',
          changeOrigin: true,
        },
      },
    },
  }
})
