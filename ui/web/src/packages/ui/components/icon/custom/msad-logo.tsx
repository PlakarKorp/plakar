import { useId } from "react";

export function MSADLogoIcon() {
  const id = useId().replace(/:/g, "");
  const gradientId = `msad-${id}-windows-gradient`;

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="16 16 96 96"
      role="img"
      aria-label="Microsoft Active Directory"
    >
      <title>Microsoft Active Directory</title>

      <defs>
        <linearGradient
          id={gradientId}
          x1="16"
          y1="20"
          x2="112"
          y2="108"
          gradientUnits="userSpaceOnUse"
        >
          <stop offset="0" stopColor="#00ADEF" />
          <stop offset="1" stopColor="#0078D4" />
        </linearGradient>
      </defs>

      <path fill={`url(#${gradientId})`} d="M16 30.5 57.5 24v38H16V30.5Z" />

      <path fill={`url(#${gradientId})`} d="M64.5 23 112 16v46H64.5V23Z" />

      <path fill={`url(#${gradientId})`} d="M16 68h41.5v38L16 99.5V68Z" />

      <path fill={`url(#${gradientId})`} d="M64.5 68H112v44l-47.5-7V68Z" />
    </svg>
  );
}
