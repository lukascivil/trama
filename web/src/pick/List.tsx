import { useLayoutEffect, useRef, useState, type ReactNode } from "react";

type Props<T> = {
  items: T[];
  rowHeight: number;
  itemKey: (item: T) => string;
  render: (item: T) => ReactNode;
};

// List draws only the rows in view, so a thousand repositories scroll and
// filter as fast as ten.
export function List<T>({ items, rowHeight, itemKey, render }: Props<T>) {
  const box = useRef<HTMLDivElement>(null);
  const [top, setTop] = useState(0);
  const [height, setHeight] = useState(600);

  useLayoutEffect(() => {
    const el = box.current;
    if (!el) return;
    const measure = () => setHeight(el.clientHeight);
    measure();
    const watcher = new ResizeObserver(measure);
    watcher.observe(el);
    return () => watcher.disconnect();
  }, []);

  const start = Math.max(0, Math.floor(top / rowHeight) - 5);
  const end = Math.min(items.length, Math.ceil((top + height) / rowHeight) + 5);
  return (
    <div className="list" ref={box} onScroll={(e) => setTop(e.currentTarget.scrollTop)} role="list">
      <div className="list-inner" style={{ height: items.length * rowHeight }}>
        {items.slice(start, end).map((item, i) => (
          <div className="list-row" role="listitem" key={itemKey(item)} style={{ top: (start + i) * rowHeight, height: rowHeight }}>
            {render(item)}
          </div>
        ))}
      </div>
    </div>
  );
}
