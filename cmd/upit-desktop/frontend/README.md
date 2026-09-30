# Upit Desktop frontend

Vue 3, TypeScript, Vite, and locally owned shadcn-vue primitives run inside Wails 3. The generated bindings under `bindings/` are refreshed with:

```bash
wails3 generate bindings ./cmd/upit-desktop -i -d ./cmd/upit-desktop/frontend/bindings
```
