import type { Metadata } from "next";
import { JetBrains_Mono, Oswald, Anton } from "next/font/google";
import "./globals.css";

const jetbrainsMono = JetBrains_Mono({
  subsets: ["latin"],
  variable: "--font-jetbrains",
  display: "swap",
});

// Sidebar: Oswald — tall, condensed, slightly rounded
const oswald = Oswald({
  weight: ["400", "500", "600"],
  subsets: ["latin"],
  variable: "--font-oswald",
  display: "swap",
});

// Header: Anton — very tight, bold, no-nonsense compressed
const anton = Anton({
  weight: "400",
  subsets: ["latin"],
  variable: "--font-anton",
  display: "swap",
});

export const metadata: Metadata = {
  title: "Launchpad",
  description: "Zero-downtime deployments via blue/green slots",
};

export default function RootLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <html lang="en" className={`dark ${jetbrainsMono.variable} ${oswald.variable} ${anton.variable}`}>
      <body className="font-mono">{children}</body>
    </html>
  );
}
