import { useId } from "react";

export function MSSQLLogoIcon() {
  const id = useId().replace(/:/g, "");
  const prefix = `mssql-${id}`;

  return (
    <svg
      xmlns="http://www.w3.org/2000/svg"
      viewBox="0 0 48 48"
      fill="none"
      role="img"
      aria-label="Microsoft SQL Server"
    >
      <title>Microsoft SQL Server</title>

      <g clipPath={`url(#${prefix}-clip0)`}>
        <g clipPath={`url(#${prefix}-clip1)`}>
          <mask
            id={`${prefix}-mask0`}
            maskUnits="userSpaceOnUse"
            x="0"
            y="0"
            width="48"
            height="48"
          >
            <path d="M48 0H0V48H48V0Z" fill="white" />
          </mask>

          <g mask={`url(#${prefix}-mask0)`}>
            <path
              d="M23.9982 33.6C34.6019 33.6 43.1982 36.8234 43.1982 40.8V26.3998C43.1982 22.4234 34.6019 19.1998 23.9982 19.1998H19.1982V33.6H23.9982Z"
              fill={`url(#${prefix}-paint0-linear)`}
            />
            <path
              d="M43.1988 26.3998C43.1988 30.3763 34.6028 33.5997 23.9988 33.5997C17.5318 33.5997 11.8116 32.4008 8.33267 30.5632C6.44883 29.728 4.79883 30.9936 4.79883 32.7091V40.7997C4.79883 44.7763 13.395 47.9997 23.9988 47.9997C34.6028 47.9997 43.1988 44.7763 43.1988 40.7997V26.3998Z"
              fill={`url(#${prefix}-paint1-radial)`}
            />
            <path
              d="M43.1988 26.3998C43.1988 30.3763 34.6028 33.5997 23.9988 33.5997C17.5318 33.5997 11.8116 32.4008 8.33267 30.5632C6.44883 29.728 4.79883 30.9936 4.79883 32.7091V40.7997C4.79883 44.7763 13.395 47.9997 23.9988 47.9997C34.6028 47.9997 43.1988 44.7763 43.1988 40.7997V26.3998Z"
              fill={`url(#${prefix}-paint2-radial)`}
            />
            <path
              d="M23.9988 0C34.6026 0 43.1989 3.22355 43.1989 7.2V15.3562C43.1989 16.4719 42.0362 18.4313 39.6621 17.4349C36.1826 15.5983 30.4639 14.4 23.9988 14.4C13.395 14.4 4.79883 17.6235 4.79883 21.6V7.2C4.79883 3.22355 13.395 0 23.9988 0Z"
              fill={`url(#${prefix}-paint3-linear)`}
            />
            <path
              d="M23.9988 14.3998C13.395 14.3998 4.79883 11.1763 4.79883 7.19983V21.5998C4.79883 25.5763 13.395 28.8 23.9988 28.8H25.1988C27.1871 28.8 28.7989 27.188 28.7989 25.1998V17.9998C28.7989 16.0116 27.1871 14.3998 25.1988 14.3998H23.9988Z"
              fill={`url(#${prefix}-paint4-radial)`}
            />
            <path
              d="M23.9988 14.3998C13.395 14.3998 4.79883 11.1763 4.79883 7.19983V21.5998C4.79883 25.5763 13.395 28.8 23.9988 28.8H25.1988C27.1871 28.8 28.7989 27.188 28.7989 25.1998V17.9998C28.7989 16.0116 27.1871 14.3998 25.1988 14.3998H23.9988Z"
              fill={`url(#${prefix}-paint5-radial)`}
            />
          </g>
        </g>
      </g>

      <defs>
        <linearGradient
          id={`${prefix}-paint0-linear`}
          x1="35.9982"
          y1="19.1998"
          x2="15.9334"
          y2="35.8194"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#0094F0" />
          <stop offset="0.243047" stopColor="#0078D4" />
          <stop offset="0.584404" stopColor="#2052CB" />
          <stop offset="0.830639" stopColor="#312A9A" />
        </linearGradient>

        <radialGradient
          id={`${prefix}-paint1-radial`}
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(33.5988 29.5499) rotate(143.591) scale(29.8211 53.0152)"
        >
          <stop stopColor="#3BD5FF" />
          <stop offset="1" stopColor="#0078D4" />
        </radialGradient>

        <radialGradient
          id={`${prefix}-paint2-radial`}
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(39.5988 27.5997) rotate(146.31) scale(25.96 43.796)"
        >
          <stop stopColor="#DECBFF" stopOpacity="0.9" />
          <stop offset="1" stopColor="#DECBFF" stopOpacity="0" />
        </radialGradient>

        <linearGradient
          id={`${prefix}-paint3-linear`}
          x1="32.3989"
          y1="1.2"
          x2="9.66835"
          y2="25.9489"
          gradientUnits="userSpaceOnUse"
        >
          <stop stopColor="#0FAFFF" />
          <stop offset="0.162714" stopColor="#0094F0" />
          <stop offset="0.563871" stopColor="#2052CB" />
          <stop offset="0.764283" stopColor="#312A9A" />
        </linearGradient>

        <radialGradient
          id={`${prefix}-paint4-radial`}
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(23.6678 12.1488) rotate(137.53) scale(24.6606 38.4117)"
        >
          <stop stopColor="#3BD5FF" />
          <stop offset="1" stopColor="#0078D4" />
        </radialGradient>

        <radialGradient
          id={`${prefix}-paint5-radial`}
          cx="0"
          cy="0"
          r="1"
          gradientUnits="userSpaceOnUse"
          gradientTransform="translate(27.5989 10.7998) rotate(149.036) scale(20.9914 33.6005)"
        >
          <stop stopColor="#DECBFF" stopOpacity="0.9" />
          <stop offset="1" stopColor="#D1D1FF" stopOpacity="0" />
        </radialGradient>

        <clipPath id={`${prefix}-clip0`}>
          <path
            d="M0 12.8C0 8.31958 0 6.07937 0.871948 4.36808C1.63893 2.86278 2.86278 1.63893 4.36808 0.871948C6.07937 0 8.31958 0 12.8 0H35.2C39.6804 0 41.9206 0 43.6319 0.871948C45.1372 1.63893 46.3611 2.86278 47.1281 4.36808C48 6.07937 48 8.31958 48 12.8V35.2C48 39.6804 48 41.9206 47.1281 43.6319C46.3611 45.1372 45.1372 46.3611 43.6319 47.1281C41.9206 48 39.6804 48 35.2 48H12.8C8.31958 48 6.07937 48 4.36808 47.1281C2.86278 46.3611 1.63893 45.1372 0.871948 43.6319C0 41.9206 0 39.6804 0 35.2V12.8Z"
            fill="white"
          />
        </clipPath>

        <clipPath id={`${prefix}-clip1`}>
          <rect width="48" height="48" fill="white" />
        </clipPath>
      </defs>
    </svg>
  );
}
