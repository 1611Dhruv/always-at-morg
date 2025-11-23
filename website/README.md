# Always at Morg - Website

A beautiful, minimalist React website for Always at Morg - the UW Madison Morgridge Hall multiplayer terminal game.

## Features

- 🎨 **Beautiful UI** - Minimalist design with UW Madison red (#c5050c) color scheme
- ✨ **Smooth Animations** - Floating badges, glowing effects, smooth transitions
- 🖱️ **Interactive** - Mouse-following gradient background, hover effects
- 📱 **Responsive** - Works great on desktop, tablet, and mobile
- 🎯 **One-page** - Easy navigation between Home, Features, and Install sections

## Development

```bash
# Install dependencies
npm install

# Start development server
npm start

# Build for production
npm run build
```

The development server will run at http://localhost:3000

## Building for Production

```bash
npm run build
```

This creates an optimized production build in the `build/` directory.

## Deployment

### Option 1: GitHub Pages

```bash
npm install -g gh-pages

# Build and deploy
npm run build
npx gh-pages -d build
```

### Option 2: Netlify/Vercel

1. Connect your repository
2. Set build command: `npm run build`
3. Set publish directory: `build`
4. Deploy!

### Option 3: Static Hosting

Simply copy the contents of `build/` to your web server.

## Customization

### Update GitHub Username

Edit `src/App.js` and replace `yourusername` with your actual GitHub username in all URLs.

### Update Server URL

Edit `src/App.js` and update the server WebSocket URL:
```javascript
morg ws://your-domain:8080/ws
```

### Change Colors

The color scheme is defined in `src/App.css`:
```css
:root {
  --uw-red: #c5050c;
  --accent: #58a6ff;
  --success: #3fb950;
}
```

## Structure

```
src/
├── App.js          # Main React component with all sections
├── App.css         # Styles and animations
├── index.js        # React entry point
└── index.css       # Global styles and scrollbar customization

public/
├── index.html      # HTML template
└── manifest.json   # PWA manifest
```

## Tech Stack

- **React 18** - UI library
- **CSS3** - Animations and styling
- **No external dependencies** - Pure React, no UI libraries needed

## License

MIT
