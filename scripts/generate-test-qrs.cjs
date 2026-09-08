// Real, clearly demo-only image fixtures. No external QR generation service.
const QRCode = require('../frontend/admin/node_modules/qrcode');
const path = require('node:path');
const fs = require('node:fs');
const dir = path.resolve(__dirname, '../workspace');
fs.mkdirSync(dir, { recursive: true });
Promise.all(['first','second'].map(name => QRCode.toFile(path.join(dir, `demo-qr-${name}.png`), `https://example.com/?gravitylink-demo=${name}`, {width:480,margin:4}))).catch(()=>process.exit(1));
