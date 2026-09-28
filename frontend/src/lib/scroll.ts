/**
 * Scrolls an element into view, smoothly unless the user prefers reduced
 * motion.
 */
export function scrollIntoView(
  element: HTMLElement | null,
  block: ScrollLogicalPosition,
) {
  const reduceMotion = window.matchMedia(
    "(prefers-reduced-motion: reduce)",
  ).matches;

  element?.scrollIntoView({
    behavior: reduceMotion ? "auto" : "smooth",
    block,
  });
}
