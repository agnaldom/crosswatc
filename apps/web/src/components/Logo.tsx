import React from "react";

interface LogoProps {
  variant?: "full" | "icon";
  size?: "sm" | "md" | "lg";
  className?: string;
  showTagline?: boolean;
}

const iconSizes = {
  sm: "h-6 w-6",
  md: "h-8 w-8",
  lg: "h-10 w-10",
};

const textSizes = {
  sm: "text-base",
  md: "text-lg",
  lg: "text-xl",
};

export function LogoIcon({ className = "h-8 w-8" }: { className?: string }) {
  return (
    <svg viewBox="0 0 128 128" fill="none" className={className} xmlns="http://www.w3.org/2000/svg" aria-label="Crosswatc Icon">
      <defs>
        <linearGradient id="cwBgGrad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#090d16" />
          <stop offset="100%" stop-color="#0f172a" />
        </linearGradient>
        <linearGradient id="cwBrandCyan" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#22d3ee" />
          <stop offset="50%" stop-color="#06b6d4" />
          <stop offset="100%" stop-color="#0284c7" />
        </linearGradient>
        <linearGradient id="cwBrandBlue" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#38bdf8" />
          <stop offset="50%" stop-color="#2563eb" />
          <stop offset="100%" stop-color="#4f46e5" />
        </linearGradient>
        <radialGradient id="cwCenterGlow" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#22d3ee" stop-opacity="0.9" />
          <stop offset="50%" stop-color="#0284c7" stop-opacity="0.4" />
          <stop offset="100%" stop-color="#0284c7" stop-opacity="0" />
        </radialGradient>
        <linearGradient id="cwBorderGrad" x1="0%" y1="0%" x2="100%" y2="100%">
          <stop offset="0%" stop-color="#38bdf8" stop-opacity="0.4" />
          <stop offset="100%" stop-color="#1e293b" stop-opacity="0.2" />
        </linearGradient>
      </defs>

      {/* Container */}
      <rect x="4" y="4" width="120" height="120" rx="28" fill="url(#cwBgGrad)" stroke="url(#cwBorderGrad)" strokeWidth="2.5" />

      {/* Radar rings */}
      <circle cx="64" cy="64" r="42" stroke="#38bdf8" strokeOpacity="0.15" strokeWidth="1.5" strokeDasharray="4 4" />
      <circle cx="64" cy="64" r="28" stroke="#38bdf8" strokeOpacity="0.2" strokeWidth="1.5" />

      {/* Radar Arc */}
      <path d="M 64 22 A 42 42 0 0 1 106 64" fill="none" stroke="url(#cwBrandCyan)" strokeWidth="3.5" strokeLinecap="round" />

      {/* Interlocking Cross Arms */}
      <path d="M 57 26 C 57 23.8 58.8 22 61 22 L 67 22 C 69.2 22 71 23.8 71 26 L 71 45 C 71 47 69 49 67 49 L 61 49 C 59 49 57 47 57 45 Z" fill="url(#cwBrandCyan)" />
      <path d="M 57 83 C 57 81 59 79 61 79 L 67 79 C 69 79 71 81 71 83 L 71 102 C 71 104.2 69.2 106 67 106 L 61 106 C 58.8 106 57 104.2 57 102 Z" fill="url(#cwBrandBlue)" />
      <path d="M 26 57 C 23.8 57 22 58.8 22 61 L 22 67 C 22 69.2 23.8 71 26 71 L 45 71 C 47 71 49 69 49 67 L 49 61 C 49 59 47 57 45 57 Z" fill="url(#cwBrandCyan)" />
      <path d="M 83 57 C 81 57 79 59 79 61 L 79 67 C 79 69 81 71 83 71 L 102 71 C 104.2 71 106 69.2 106 67 L 106 61 C 106 58.8 104.2 57 102 57 Z" fill="url(#cwBrandBlue)" />

      {/* Center Pulse Core */}
      <circle cx="64" cy="64" r="16" fill="url(#cwCenterGlow)" />
      <circle cx="64" cy="64" r="8" fill="#0f172a" stroke="url(#cwBrandCyan)" strokeWidth="2.5" />
      <circle cx="64" cy="64" r="3.5" fill="#38bdf8" />

      {/* Corner Brackets */}
      <path d="M 38 32 L 32 32 L 32 38" stroke="#38bdf8" strokeOpacity="0.45" strokeWidth="2" strokeLinecap="round" />
      <path d="M 90 32 L 96 32 L 96 38" stroke="#38bdf8" strokeOpacity="0.45" strokeWidth="2" strokeLinecap="round" />
      <path d="M 38 96 L 32 96 L 32 90" stroke="#38bdf8" strokeOpacity="0.45" strokeWidth="2" strokeLinecap="round" />
      <path d="M 90 96 L 96 96 L 96 90" stroke="#38bdf8" strokeOpacity="0.45" strokeWidth="2" strokeLinecap="round" />
    </svg>
  );
}

export function Logo({
  variant = "full",
  size = "md",
  className = "",
  showTagline = false,
}: LogoProps) {
  if (variant === "icon") {
    return <LogoIcon className={`${iconSizes[size]} ${className}`} />;
  }

  return (
    <div className={`flex items-center gap-2.5 ${className}`}>
      <LogoIcon className={`${iconSizes[size]} shrink-0`} />
      <div className="flex flex-col">
        <span className={`font-bold tracking-tight text-zinc-900 ${textSizes[size]}`}>
          Cross<span className="bg-gradient-to-r from-cyan-500 to-blue-600 bg-clip-text text-transparent">watc</span>
        </span>
        {showTagline && (
          <span className="text-[10px] font-medium uppercase tracking-wider text-zinc-400">
            GitOps Lifecycle
          </span>
        )}
      </div>
    </div>
  );
}
