// Wires up arrow buttons + edge-aware visibility for each .carousel on the page.
// Native horizontal scroll handles touch/swipe; this script just adds desktop
// arrow controls and toggles them when the user hits either end.

function initCarousel(root: HTMLElement) {
  const viewport = root.querySelector<HTMLElement>("[data-carousel-viewport]");
  const prev = root.querySelector<HTMLButtonElement>("[data-carousel-prev]");
  const next = root.querySelector<HTMLButtonElement>("[data-carousel-next]");
  if (!viewport || !prev || !next) return;

  // Scroll by ~80% of viewport width so the user gets a fresh page of cards
  // each click while keeping a small overlap for context.
  function scrollDelta(): number {
    return Math.max(160, Math.floor(viewport!.clientWidth * 0.8));
  }

  function updateArrows() {
    const max = viewport!.scrollWidth - viewport!.clientWidth;
    const x = viewport!.scrollLeft;
    // Tolerance for sub-pixel rounding — at exact edges browsers may report
    // 0.5px off.
    const atStart = x <= 1;
    const atEnd = x >= max - 1;
    // Hide both arrows when the track fits in one viewport (nothing to scroll).
    const overflows = max > 1;
    prev!.hidden = !overflows || atStart;
    next!.hidden = !overflows || atEnd;
  }

  prev.addEventListener("click", () => {
    viewport.scrollBy({ left: -scrollDelta(), behavior: "smooth" });
  });
  next.addEventListener("click", () => {
    viewport.scrollBy({ left: scrollDelta(), behavior: "smooth" });
  });

  viewport.addEventListener("scroll", updateArrows, { passive: true });

  // ResizeObserver picks up width changes (window resize, font load, etc.) so
  // we re-evaluate whether arrows are needed.
  const ro = new ResizeObserver(updateArrows);
  ro.observe(viewport);
  // Also recheck after images load — late-arriving images can change scrollWidth.
  viewport.querySelectorAll("img").forEach((img) => {
    if (!img.complete) img.addEventListener("load", updateArrows, { once: true });
  });

  updateArrows();
}

export function initCarousels() {
  const carousels = document.querySelectorAll<HTMLElement>(".carousel");
  carousels.forEach((c) => {
    if (c.dataset.carouselInited === "1") return;
    c.dataset.carouselInited = "1";
    initCarousel(c);
  });
}
