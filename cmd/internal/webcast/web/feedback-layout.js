// Geometry only: the floating-panel owner applies these bounds and owns dragging.
export function feedbackLayout({
  width, height, toolbarWidth = 56, gap = 12, panelWidth = 300, panelHeight = 320,
}) {
  const nonnegative = value => Number.isFinite(value) ? Math.max(0, value) : 0;
  width = nonnegative(width);
  height = nonnegative(height);
  toolbarWidth = nonnegative(toolbarWidth);
  gap = nonnegative(gap);
  panelWidth = nonnegative(panelWidth);
  panelHeight = nonnegative(panelHeight);

  // Shrinking margins leaves even a zero-size viewport with valid coordinates.
  const marginX = Math.min(gap, width / 2);
  const marginY = Math.min(gap, height / 2);
  const innerWidth = width - marginX - marginX;
  const innerHeight = height - marginY - marginY;
  const reservedWidth = Math.min(toolbarWidth, innerWidth);
  const toolbarLeft = Math.max(marginX, width - marginX - reservedWidth);
  const contentRight = Math.max(marginX, toolbarLeft - gap);
  const contentWidth = contentRight - marginX;
  const fittedWidth = Math.min(panelWidth, contentWidth);
  const fittedHeight = Math.min(panelHeight, innerHeight);

  return {
    toolbar: { left: toolbarLeft, top: marginY, width: reservedWidth, height: innerHeight },
    content: { left: marginX, top: marginY, width: contentWidth, height: innerHeight },
    panel: { width: fittedWidth, height: fittedHeight },
    initial: {
      left: Math.max(marginX, contentRight - fittedWidth),
      top: Math.max(marginY, Math.min(110, height - marginY - fittedHeight)),
    },
  };
}
