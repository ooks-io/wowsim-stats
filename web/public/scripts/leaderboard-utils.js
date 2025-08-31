export function formatDuration(milliseconds) {
  const totalSeconds = Math.floor(milliseconds / 1000);
  const minutes = Math.floor(totalSeconds / 60);
  const seconds = totalSeconds % 60;
  return minutes + ":" + seconds.toString().padStart(2, "0");
}

export function formatTimestamp(timestamp) {
  return new Date(timestamp).toLocaleDateString("en-US", {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  });
}

window.toggleChartItem = (element) => {
  const wrapper = element.closest(".chart-item-wrapper");
  if (!wrapper) return;

  const isExpanded = wrapper.classList.contains("chart-item-expanded");

  document.querySelectorAll(".chart-item-expanded").forEach((item) => {
    if (item !== wrapper) {
      item.classList.remove("chart-item-expanded");
    }
  });

  wrapper.classList.toggle("chart-item-expanded", !isExpanded);
};
