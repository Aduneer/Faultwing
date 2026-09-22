import { useId } from "react";

type BrandMarkProps = {
  className?: string;
};

const markPath = "M1976 10578c-12-75-13-120-5-255 37-653 323-1247 794-1651 465-399 1115-648 2415-926 91-19 273-58 405-86 629-134 862-192 1105-275 161-55 365-156 473-233 289-208 466-531 484-886l6-109-31 22c-17 12-34 26-37 31-3 6-50 41-103 79-294 208-691 383-1237 546-258 76-483 135-1145 299-880 218-1183 308-1605 477-237 96-594 286-804 429-33 22-62 40-65 40-23 0 20-202 85-397 48-145 153-361 239-493 228-347 537-601 955-785 363-159 653-242 1590-455 665-151 944-232 1193-351 279-132 475-314 588-546 72-148 114-319 114-466v-68l-60 64c-162 173-481 334-900 452-143 41-525 137-875 220-859 205-1371 384-1755 611-97 58-152 95-318 211-34 24-63 43-66 43-14 0-4-75 23-171 55-194 136-391 268-654 467-932 1051-1655 1838-2276 239-189 519-385 988-695 234-154 279-174 382-174 117 0 144 13 470 230 839 557 1566 1204 1998 1777 341 453 639 1069 781 1615 151 580 198 1056 212 2173 6 401 4 469-10 527-39 166-147 304-290 373-55 26-119 45-245 70-382 78-622 142-967 259-318 108-533 196-829 342-293 144-424 218-695 396-162 105-220 128-327 128-102-1-143-16-343-127-376-207-836-406-1287-554-73-24-131-45-129-47 2-2 86-25 187-52 564-148 874-245 1199-373 484-190 783-389 981-651 111-147 194-347 205-493l5-78-78 150c-234 449-492 680-1013 904-417 180-902 326-1915 575-500 124-719 185-972 270-657 223-1191 503-1618 847-66 54-142 120-170 148-58 60-82 65-89 19z";

export function BrandMark({ className = "" }: BrandMarkProps) {
  const id = useId();
  const shapeId = `${id}-faultwing-shape`;
  const fillId = `${id}-faultwing-fill`;
  const sheenId = `${id}-faultwing-sheen`;

  return (
    <svg
      className={`brand-mark ${className}`.trim()}
      viewBox="160 160 920 920"
      aria-hidden="true"
      focusable="false"
    >
      <defs>
        <path id={shapeId} d={markPath} />
        <linearGradient id={fillId} x1="0" y1="0" x2="1" y2="1">
          <stop offset="0" stopColor="#64c970" />
          <stop offset="0.43" stopColor="#2d984c" />
          <stop offset="1" stopColor="#176a36" />
        </linearGradient>
        <linearGradient id={sheenId} x1="0" y1="0" x2="0.72" y2="0.88">
          <stop offset="0" stopColor="#ffffff" stopOpacity="0.48" />
          <stop offset="0.3" stopColor="#ffffff" stopOpacity="0.12" />
          <stop offset="0.58" stopColor="#ffffff" stopOpacity="0" />
        </linearGradient>
      </defs>
      <g transform="translate(0 1254) scale(.1 -.1)">
        <use className="brand-mark__base" href={`#${shapeId}`} fill={`url(#${fillId})`} />
        <use
          className="brand-mark__sheen"
          href={`#${shapeId}`}
          fill={`url(#${sheenId})`}
          stroke="rgba(255, 255, 255, 0.34)"
          strokeWidth="15"
        />
      </g>
    </svg>
  );
}
