const fs = require('fs');
const path = require('path');

function generateLogoSvg() {
  const cDarkest = '#070b12';
  const cDark = '#0f172a';
  const cShadow = '#1e293b';
  const cMid = '#334155';
  const cLight = '#475569';
  const cSlate = '#64748b';
  const cSilver = '#94a3b8';
  const cCyanBright = '#00f5ff';
  const cCyanDeep = '#0284c7';

  return `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 1024 1024" width="1024" height="1024">
  <defs>
    <clipPath id="squircle-clip">
      <rect x="24" y="24" width="976" height="976" rx="220" />
    </clipPath>

    <linearGradient id="cyan-glow" x1="0%" y1="0%" x2="100%" y2="100%">
      <stop offset="0%" stop-color="#00f5ff"/>
      <stop offset="100%" stop-color="#0284c7"/>
    </linearGradient>

    <filter id="subtle-shadow" x="-20%" y="-20%" width="140%" height="140%">
      <feDropShadow dx="0" dy="16" stdDeviation="20" flood-color="#000000" flood-opacity="0.18" />
    </filter>
  </defs>

  <!-- Luxury White Squircle Container -->
  <rect x="24" y="24" width="976" height="976" rx="220" fill="#ffffff" stroke="#e2e8f0" stroke-width="6" />

  <g clip-path="url(#squircle-clip)">
    <g transform="translate(512, 512)" filter="url(#subtle-shadow)">

      <!-- Hexagonal Architectural Gateway -->
      <polygon points="
        0,-390
        338,-195
        338,195
        0,390
        -338,195
        -338,-195
      " fill="none" stroke="#0f172a" stroke-width="40" stroke-linejoin="round" />

      <!-- Inner Precision Telemetry Hexagon -->
      <polygon points="
        0,-355
        307,-177
        307,177
        0,355
        -307,177
        -307,-177
      " fill="none" stroke="#00f5ff" stroke-width="4" opacity="0.4" stroke-dasharray="16, 12" stroke-linejoin="round" />

      <!-- Acoustic Sonar Echo Waveforms -->
      <path d="M -190,-295 A 350 350 0 0 1 190,-295" fill="none" stroke="#00f5ff" stroke-width="2.5" opacity="0.3" stroke-dasharray="10, 8" />
      <path d="M -135,-325 A 360 360 0 0 1 135,-325" fill="none" stroke="#00f5ff" stroke-width="1.5" opacity="0.2" stroke-dasharray="6, 6" />

      <!-- Watertight Base Silhouette -->
      <path d="
        M 0,-155
        L 32,-158 L 115,-275 L 145,-140 L 215,-90 L 305,-175 L 325,-120 L 280,-15 L 290,65 L 235,135 L 215,205 L 145,180 L 85,250 L 0,280
        L -85,250 L -145,180 L -215,205 L -235,135 L -290,65 L -280,-15 L -325,-120 L -305,-175 L -215,-90 L -145,-140 L -115,-275 L -32,-158 Z
      " fill="${cDarkest}" />

      <!-- ==================== WINGS ==================== -->
      <!-- Right Wing Facets -->
      <polygon points="145,-140 215,-90 305,-175 260,-130" fill="${cMid}" />
      <polygon points="305,-175 325,-120 280,-15 260,-65" fill="${cShadow}" />
      <polygon points="260,-130 305,-175 260,-65" fill="${cDark}" />
      <polygon points="280,-15 290,65 220,35" fill="${cDarkest}" />
      <polygon points="220,35 290,65 235,135 185,95" fill="${cShadow}" />
      <polygon points="185,95 235,135 215,205 145,180" fill="${cDarkest}" />
      <polygon points="145,180 145,120 185,95" fill="${cMid}" />
      <polygon points="260,-130 280,-15 220,35 215,-90" fill="${cMid}" />
      <polygon points="215,-90 220,35 145,25 145,-140" fill="${cShadow}" />
      <polygon points="145,25 220,35 185,95 145,120" fill="${cLight}" />

      <!-- Left Wing Facets -->
      <polygon points="-145,-140 -215,-90 -305,-175 -260,-130" fill="${cLight}" />
      <polygon points="-305,-175 -325,-120 -280,-15 -260,-65" fill="${cDark}" />
      <polygon points="-260,-130 -305,-175 -260,-65" fill="${cShadow}" />
      <polygon points="-280,-15 -290,65 -220,35" fill="${cShadow}" />
      <polygon points="-220,35 -290,65 -235,135 -185,95" fill="${cMid}" />
      <polygon points="-185,95 -235,135 -215,205 -145,180" fill="${cDark}" />
      <polygon points="-145,180 -145,120 -185,95" fill="${cLight}" />
      <polygon points="-260,-130 -280,-15 -220,35 -215,-90" fill="${cLight}" />
      <polygon points="-215,-90 -220,35 -145,25 -145,-140" fill="${cMid}" />
      <polygon points="-145,25 -220,35 -185,95 -145,120" fill="${cShadow}" />

      <!-- Cyan Wing Leading Edge Accents -->
      <polyline points="145,-140 215,-90 305,-175 325,-120" fill="none" stroke="${cCyanBright}" stroke-width="4.5" stroke-linecap="round" stroke-linejoin="round" />
      <polyline points="-145,-140 -215,-90 -305,-175 -325,-120" fill="none" stroke="${cCyanBright}" stroke-width="4.5" stroke-linecap="round" stroke-linejoin="round" />

      <!-- ==================== EARS (ULTRASONIC RADAR HORNS) ==================== -->
      <!-- Right Ear -->
      <polygon points="32,-158 115,-275 75,-165" fill="${cShadow}" />
      <polygon points="75,-165 115,-275 145,-140" fill="${cDark}" />
      <polygon points="40,-155 105,-255 75,-170" fill="${cCyanDeep}" />
      <polygon points="75,-170 105,-255 125,-150" fill="${cCyanBright}" opacity="0.9" />
      <polygon points="55,-160 95,-235 78,-175" fill="#ffffff" opacity="0.55" />

      <!-- Left Ear -->
      <polygon points="-32,-158 -115,-275 -75,-165" fill="${cLight}" />
      <polygon points="-75,-165 -115,-275 -145,-140" fill="${cMid}" />
      <polygon points="-40,-155 -105,-255 -75,-170" fill="${cCyanBright}" opacity="0.9" />
      <polygon points="-75,-170 -105,-255 -125,-150" fill="${cCyanDeep}" />
      <polygon points="-55,-160 -95,-235 -78,-175" fill="#ffffff" opacity="0.55" />

      <!-- Forehead Crown -->
      <polygon points="0,-155 32,-158 75,-165 0,-130" fill="${cMid}" />
      <polygon points="0,-155 -32,-158 -75,-165 0,-130" fill="${cLight}" />

      <!-- ==================== FACIAL ARCHITECTURE ==================== -->
      <!-- Upper Brow & Temples -->
      <polygon points="0,-130 75,-165 85,-105 0,-95" fill="${cShadow}" />
      <polygon points="0,-130 -75,-165 -85,-105 0,-95" fill="${cMid}" />

      <!-- Zygomatic Cheek Flanges -->
      <polygon points="85,-105 145,-140 145,25 75,-30" fill="${cDark}" />
      <polygon points="-85,-105 -145,-140 -145,25 -75,-30" fill="${cShadow}" />

      <!-- Upper Brow Plate -->
      <polygon points="0,-95 45,-90 35,-65 0,-60" fill="${cMid}" />
      <polygon points="0,-95 -45,-90 -35,-65 0,-60" fill="${cLight}" />

      <!-- PREDATORY SONAR OPTICS (Sharp geometric almond eyes) -->
      <!-- Right Eye Socket -->
      <polygon points="35,-82 78,-74 58,-60 28,-68" fill="${cDarkest}" />
      <polygon points="38,-80 74,-74 56,-62 31,-67" fill="url(#cyan-glow)" />
      <polygon points="46,-76 66,-73 54,-64 42,-67" fill="#ffffff" />

      <!-- Left Eye Socket -->
      <polygon points="-35,-82 -78,-74 -58,-60 -28,-68" fill="${cDarkest}" />
      <polygon points="-38,-80 -74,-74 -56,-62 -31,-67" fill="url(#cyan-glow)" />
      <polygon points="-46,-76 -66,-73 -54,-64 -42,-67" fill="#ffffff" />

      <!-- BAT NOSE-LEAF & SNOUT (Characteristic ultrasonic echolocation leaf) -->
      <!-- Nose Leaf Crest (Upward triangular sonar blade) -->
      <polygon points="0,-60 18,-45 0,-30" fill="${cLight}" />
      <polygon points="0,-60 -18,-45 0,-30" fill="${cSilver}" />
      <polygon points="0,-55 12,-43 0,-32" fill="${cCyanBright}" opacity="0.8" />
      <polygon points="0,-55 -12,-43 0,-32" fill="${cCyanDeep}" opacity="0.8" />

      <!-- Lateral Snout Flanges -->
      <polygon points="18,-45 75,-30 45,-5 0,-10" fill="${cShadow}" />
      <polygon points="-18,-45 -75,-30 -45,-5 0,-10" fill="${cMid}" />
      <polygon points="0,-30 18,-45 0,-10" fill="${cSlate}" />
      <polygon points="0,-30 -18,-45 0,-10" fill="${cSilver}" />

      <!-- Aerodynamic Chin & Jaw -->
      <polygon points="0,-10 45,-5 25,25 0,32" fill="${cMid}" />
      <polygon points="0,-10 -45,-5 -25,25 0,32" fill="${cLight}" />
      <polygon points="45,-5 75,-30 65,20 25,25" fill="${cDark}" />
      <polygon points="-45,-5 -75,-30 -65,20 -25,25" fill="${cShadow}" />

      <!-- ==================== TORSO & CASSETTE MECHANISM ==================== -->
      <!-- Upper Chest Plate -->
      <polygon points="0,32 65,20 110,65 65,140 0,165" fill="${cShadow}" />
      <polygon points="0,32 -65,20 -110,65 -65,140 0,165" fill="${cMid}" />
      <polygon points="110,65 145,25 145,120 65,140" fill="${cDark}" />
      <polygon points="-110,65 -145,25 -145,120 -65,140" fill="${cShadow}" />

      <!-- Lower Body & Tail Membrane -->
      <polygon points="0,165 65,140 85,250 0,280" fill="${cDarkest}" />
      <polygon points="0,165 -65,140 -85,250 0,280" fill="${cDark}" />
      <polygon points="65,140 145,180 85,250" fill="${cDark}" />
      <polygon points="-65,140 -145,180 -85,250" fill="${cMid}" />

      <!-- CASSETTE SPOOL HUBS & AUDIO TAPE GUIDELINE -->
      <!-- Tape Bridge Line -->
      <line x1="-52" y1="92" x2="52" y2="92" stroke="${cCyanBright}" stroke-width="4" stroke-dasharray="6, 4" opacity="0.85" />

      <!-- Left Reel Hub (Center at -52, 92) -->
      <circle cx="-52" cy="92" r="28" fill="${cDarkest}" stroke="${cCyanBright}" stroke-width="3.5" />
      <circle cx="-52" cy="92" r="14" fill="${cMid}" stroke="${cCyanBright}" stroke-width="2" />
      <circle cx="-52" cy="92" r="5" fill="${cCyanBright}" />
      <!-- Left Reel Spokes -->
      <line x1="-52" y1="68" x2="-52" y2="76" stroke="#ffffff" stroke-width="3" stroke-linecap="round" />
      <line x1="-31" y1="104" x2="-38" y2="100" stroke="#ffffff" stroke-width="3" stroke-linecap="round" />
      <line x1="-73" y1="104" x2="-66" y2="100" stroke="#ffffff" stroke-width="3" stroke-linecap="round" />

      <!-- Right Reel Hub (Center at +52, 92) -->
      <circle cx="52" cy="92" r="28" fill="${cDarkest}" stroke="${cCyanBright}" stroke-width="3.5" />
      <circle cx="52" cy="92" r="14" fill="${cMid}" stroke="${cCyanBright}" stroke-width="2" />
      <circle cx="52" cy="92" r="5" fill="${cCyanBright}" />
      <!-- Right Reel Spokes -->
      <line x1="52" y1="68" x2="52" y2="76" stroke="#ffffff" stroke-width="3" stroke-linecap="round" />
      <line x1="31" y1="104" x2="38" y2="100" stroke="#ffffff" stroke-width="3" stroke-linecap="round" />
      <line x1="73" y1="104" x2="66" y2="100" stroke="#ffffff" stroke-width="3" stroke-linecap="round" />

      <!-- Sub-millisecond Replay Pulse Diamond -->
      <polygon points="0,50 10,62 0,74 -10,62" fill="${cCyanBright}" />
      <circle cx="0" cy="62" r="3" fill="#ffffff" />

      <!-- Stealth Spine Highlight -->
      <line x1="0" y1="165" x2="0" y2="280" stroke="${cCyanBright}" stroke-width="3" opacity="0.65" stroke-linecap="round" />

    </g>
  </g>
</svg>`;
}

const svgContent = generateLogoSvg();
const docsImages = path.join(__dirname, '..', 'docs', 'images');
fs.writeFileSync(path.join(docsImages, 'logo.svg'), svgContent, 'utf8');
console.log('✓ Wrote refined docs/images/logo.svg successfully.');
