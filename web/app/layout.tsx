import type { Metadata } from 'next';
import { Figtree } from 'next/font/google';
import localFont from 'next/font/local';
import './globals.css';

const figtree = Figtree({ subsets: ['latin'], variable: '--font-figtree' });

// Go Mono by Bigelow & Holmes, made for the Go project (BSD license, see
// app/fonts/GO-FONTS-LICENSE.txt).
const goMono = localFont({
  src: [
    { path: './fonts/Go-Mono.ttf', weight: '400' },
    { path: './fonts/Go-Mono-Bold.ttf', weight: '700' },
  ],
  variable: '--font-go-mono',
});

export const metadata: Metadata = {
  title: 'Shipyard: a go-ai demo',
  description:
    'useChat in the browser, a Go backend built with the Go AI SDK, tool approvals, and Claude Code or Codex doing the coding.',
  icons: { icon: '/logo.png' },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="en" className={`${figtree.variable} ${goMono.variable}`}>
      <body>{children}</body>
    </html>
  );
}
