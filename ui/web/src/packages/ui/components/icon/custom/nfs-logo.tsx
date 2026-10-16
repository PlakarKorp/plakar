export function NFSLogoIcon() {
  return (
    <svg viewBox="0 0 128 128" role="img" aria-label="NFS">
      <title>NFS</title>

      <defs>
        <mask id="nfs-logo-mask">
          <rect width="128" height="128" fill="white" />
          <text
            x="65"
            y="80"
            textAnchor="middle"
            fontFamily="Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, 'Segoe UI', sans-serif"
            fontSize="44"
            fontWeight="800"
            letterSpacing="1.5"
            fill="black"
          >
            NFS
          </text>
        </mask>
      </defs>

      <path
        fill="currentColor"
        mask="url(#nfs-logo-mask)"
        d="M15.156 16.844C11.063 16.844 7.75 20.157 7.75 24.25v64.875c0 4.093 3.313 7.375 7.406 7.375h43v4.656h-5.937c-1.439 0-2.594 1.155-2.594 2.594v4.813c0 1.438 1.155 2.593 2.594 2.593h23.562c1.439 0 2.594-1.155 2.594-2.593v-4.813c0-1.439-1.155-2.594-2.594-2.594h-5.937V96.5h43c4.093 0 7.406-3.282 7.406-7.375v-51.75c0-4.093-3.313-7.35-7.406-7.375H53.406v-5.75c0-4.093-3.281-7.406-7.375-7.406H15.156zM7.75 102.906v6.5h39.594v-5.562c0-.32.038-.635.094-.938H7.75zm72.813 0c.055.303.093.618.093.938v5.562h39.594v-6.5H80.563z"
      />
    </svg>
  );
}
