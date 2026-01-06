import {defineConfig} from "vite";
import react from "@vitejs/plugin-react";

export default defineConfig({
  plugins: [react()],
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          react: ['react', 'react-dom', 'react-router', 'react-hook-form', 'react-i18next', 'react-input-mask'],
          emotion: ['@emotion/react', '@emotion/styled'],
          refine: ['@refinedev/cli', '@refinedev/core', '@refinedev/kbar', '@refinedev/mui','@mui/lab', '@mui/material', '@mui/x-data-grid', '@mui/icons-material'],
          extra: ['@tanstack/react-table', '@uiw/react-md-editor'],
          tools: ['axios', 'lodash', 'dayjs'],
          googlemaps: ['@googlemaps/react-wrapper', 'google-map-react'],
        },
      },
    }
  }
});
