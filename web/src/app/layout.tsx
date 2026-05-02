import "./globals.css";
import { ThemeProvider } from "@/provider/theme";
import { Toaster } from "@/components/ui/sonner"
import { LocaleProvider } from "@/provider/locale";
import QueryProvider from "@/provider/query";
import { ServiceWorkerRegister } from "@/components/sw-register";
import { TooltipProvider } from "@/components/animate-ui/components/animate/tooltip";



export default function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <html suppressHydrationWarning>
      <head>
        <meta name="theme-color" content="#eae9e3" />
        <meta name="application-name" content="Octopus" />
        <meta name="apple-mobile-web-app-capable" content="yes" />
        <meta name="apple-mobile-web-app-status-bar-style" content="black" />
        <meta name="apple-mobile-web-app-title" content="Octopus" />
        <meta name="mobile-web-app-capable" content="yes" />
        <meta name="mobile-web-app-status-bar-style" content="black" />
        <meta name="mobile-web-app-title" content="Octopus" />
        <link rel="manifest" href="./manifest.json" />
        <link rel="icon" href="./favicon.ico" sizes="any" />
        <link rel="apple-touch-icon" href="./apple-icon.png" />
        <title>Octopus</title>
        <style
          dangerouslySetInnerHTML={{
            __html: `
              #initial-loader {
                position: fixed;
                inset: 0;
                z-index: 9999;
                display: flex;
                align-items: center;
                justify-content: center;
                overflow: hidden;
                background: linear-gradient(180deg, rgba(234, 238, 232, 0.98) 0%, rgba(225, 230, 221, 0.98) 100%);
                color: var(--primary);
                transition: opacity 280ms ease;
              }
              #initial-loader.octo-hide {
                opacity: 0;
                pointer-events: none;
              }
              #initial-loader::before,
              #initial-loader::after {
                content: '';
                position: absolute;
                pointer-events: none;
                border-radius: 999px;
                filter: blur(48px);
              }
              #initial-loader::before {
                width: 18rem;
                height: 18rem;
                top: 8%;
                left: 10%;
                background: rgba(248, 251, 247, 0.72);
                animation: octoMist 7s ease-in-out infinite;
              }
              #initial-loader::after {
                width: 22rem;
                height: 22rem;
                right: 6%;
                bottom: -6%;
                background: rgba(136, 200, 188, 0.16);
                animation: octoDrift 10s ease-in-out infinite;
              }
              #initial-loader .octo-shell {
                position: relative;
                display: grid;
                place-items: center;
                width: min(13rem, 40vw);
                aspect-ratio: 1;
                border-radius: 32%;
                border: 1px solid rgba(177, 191, 181, 0.48);
                background: linear-gradient(180deg, rgba(255, 255, 255, 0.84) 0%, rgba(244, 248, 243, 0.68) 100%);
                box-shadow: var(--waterhouse-shadow-soft);
              }
              #initial-loader .octo-shell::before {
                content: '';
                position: absolute;
                inset: 0.75rem;
                border-radius: 28%;
                border: 1px solid color-mix(in oklch, white 16%, transparent);
                opacity: 0.7;
              }
              @supports (background: color-mix(in oklch, white 50%, transparent)) {
                #initial-loader {
                  background:
                    radial-gradient(circle at 20% 18%, color-mix(in oklch, var(--waterhouse-highlight) 16%, transparent) 0%, transparent 28%),
                    radial-gradient(circle at 80% 16%, color-mix(in oklch, var(--primary) 12%, transparent) 0%, transparent 26%),
                    linear-gradient(180deg, var(--waterhouse-bg-top) 0%, var(--background) 46%, var(--waterhouse-bg-bottom) 100%);
                }
                #initial-loader::before {
                  background: color-mix(in oklch, var(--waterhouse-fog-strong) 78%, transparent);
                }
                #initial-loader::after {
                  background: color-mix(in oklch, var(--waterhouse-highlight) 14%, transparent);
                }
                #initial-loader .octo-shell {
                  border-color: var(--waterhouse-line);
                  background: linear-gradient(180deg, var(--waterhouse-fog-strong) 0%, var(--waterhouse-fog) 100%);
                }
              }
              @supports ((backdrop-filter: blur(1px)) or (-webkit-backdrop-filter: blur(1px))) {
                #initial-loader {
                  backdrop-filter: blur(12px);
                }
                #initial-loader .octo-shell {
                  backdrop-filter: blur(22px);
                  -webkit-backdrop-filter: blur(22px);
                }
              }
              #initial-loader svg {
                position: relative;
                width: 7rem;
                height: 7rem;
              }
              #initial-loader .octo-group {
                animation: octoFade 2s ease-in-out infinite;
              }
              #initial-loader path {
                fill: none;
                stroke: currentColor;
                stroke-width: 6;
                stroke-linecap: round;
                stroke-dasharray: 1;
                stroke-dashoffset: 1;
                opacity: 0;
                animation: octoDraw 2s ease-in-out infinite both;
              }
              #initial-loader path:nth-child(1) { animation-delay: 0s; }
              #initial-loader path:nth-child(2) { animation-delay: 0.15s; }
              #initial-loader path:nth-child(3) { animation-delay: 0.30s; }
              #initial-loader path:nth-child(4) { animation-delay: 0.45s; }
              #initial-loader path:nth-child(5) { animation-delay: 0.60s; }

              @keyframes octoDraw {
                0%   { stroke-dashoffset: 1; opacity: 0; }
                5%   { opacity: 1; }
                40%  { stroke-dashoffset: 0; opacity: 1; }
                100% { stroke-dashoffset: 0; opacity: 1; }
              }
              @keyframes octoFade {
                0%   { opacity: 1; }
                70%  { opacity: 1; }
                100% { opacity: 0; }
              }
              @keyframes octoMist {
                0%, 100% { transform: translate3d(0, 0, 0) scale(1); opacity: 0.72; }
                50% { transform: translate3d(3%, 4%, 0) scale(1.08); opacity: 1; }
              }
              @keyframes octoDrift {
                0%, 100% { transform: translate3d(0, 0, 0) scale(1); }
                50% { transform: translate3d(-4%, -3%, 0) scale(1.04); }
              }

              @media (prefers-reduced-motion: reduce) {
                #initial-loader::before,
                #initial-loader::after,
                #initial-loader .octo-group,
                #initial-loader path {
                  animation: none !important;
                  opacity: 1 !important;
                  stroke-dashoffset: 0 !important;
                }
              }
            `,
          }}
        />
      </head>
      <body className="waterhouse-shell antialiased">
        {/* ── Nature / Waterhouse: SVG 滤镜定义（全局可用） ── */}
        <svg style={{ visibility: "hidden", position: "absolute" }} width="0" height="0" aria-hidden="true">
          <defs>
            {/* 液态融合滤镜——Nature 风格标志性效果 */}
            <filter id="nature-gooey">
              <feGaussianBlur in="SourceGraphic" stdDeviation="10" result="blur" />
              <feColorMatrix
                in="blur"
                mode="matrix"
                values="1 0 0 0 0  0 1 0 0 0  0 0 1 0 0  0 0 0 19 -9"
                result="gooey"
              />
              <feComposite in="SourceGraphic" in2="gooey" operator="atop" />
            </filter>
            {/* 柔和弥散光 */}
            <filter id="nature-soft-glow">
              <feGaussianBlur in="SourceGraphic" stdDeviation="6" result="blur" />
              <feMerge>
                <feMergeNode in="blur" />
                <feMergeNode in="SourceGraphic" />
              </feMerge>
            </filter>
          </defs>
        </svg>
        <div className="waterhouse-canopy" aria-hidden="true" />
        <div className="waterhouse-liquid-field" aria-hidden="true" />
        {/* ── Nature: 第三个环境 Blob（中景） ── */}
        <div className="nature-ambient-blob" aria-hidden="true" />
        <div id="initial-loader" role="status" aria-label="Loading">
          <div className="octo-shell">
            <svg viewBox="0 0 100 100" xmlns="http://www.w3.org/2000/svg">
              <g className="octo-group">
                <path pathLength="1" d="M50 15 C70 15 85 30 85 50 C85 65 75 75 70 80 M50 15 C30 15 15 30 15 50 C15 65 25 75 30 80" />
                <path pathLength="1" d="M30 80 Q30 90 20 90" />
                <path pathLength="1" d="M43 77 Q43 90 38 90" />
                <path pathLength="1" d="M57 77 Q57 90 62 90" />
                <path pathLength="1" d="M70 80 Q70 90 80 90" />
              </g>
            </svg>
          </div>
        </div>
        <ServiceWorkerRegister />
        <ThemeProvider attribute="class" defaultTheme="system" enableSystem>
          <QueryProvider>
            <LocaleProvider>
              <TooltipProvider>
                {children}
                <Toaster />
              </TooltipProvider>
            </LocaleProvider>
          </QueryProvider>
        </ThemeProvider>
      </body>
    </html>
  );
}
