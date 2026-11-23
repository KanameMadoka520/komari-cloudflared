import type { Transition } from "motion/react";

export const MOTION_DURATION = { fast: 0.12 } as const;

export const MOTION_EASE = { standard: [0.2, 0, 0, 1] } as const;

export const MOTION_SPRING = {
  layout: { type: "spring", stiffness: 560, damping: 42, mass: 0.75 },
  snappy: { type: "spring", stiffness: 820, damping: 40, mass: 0.5 },
  soft: { type: "spring", stiffness: 380, damping: 34, mass: 0.85 },
} satisfies Record<string, Transition>;

export const MOTION_TRANSITION = {
  fade: { duration: MOTION_DURATION.fast, ease: MOTION_EASE.standard },
} satisfies Record<string, Transition>;

export const CONTENT_MOTION = {
  item: {
    initial: { opacity: 0, y: 12, scale: 0.985 },
    animate: { opacity: 1, y: 0, scale: 1 },
    exit: { opacity: 0, scale: 0.97 },
  },
} as const;

export const CONTROL_MOTION = {
  tap: { scaleX: 1.02, scaleY: 0.96 },
  transition: MOTION_SPRING.snappy,
} as const;

export const FEEDBACK_MOTION = {
  number: { duration: 0.18, ease: [0.22, 1, 0.36, 1] },
} satisfies Record<string, Transition>;
