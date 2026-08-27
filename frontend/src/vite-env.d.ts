/// <reference types="vite/client" />

export {};

declare module "vue-router" {
  interface RouteMeta {
    title?: string;
  }
}

