import { useEffect, useRef, useState } from 'react';
import { createPortal } from 'react-dom';

export interface DropdownMenuItem {
  label: string;
  icon?: React.ReactNode;
  onClick: () => void;
  variant?: 'default' | 'danger';
}

export interface DropdownMenuProps {
  items: DropdownMenuItem[];
  isOpen: boolean;
  onClose: () => void;
  triggerRef: React.RefObject<HTMLElement>;
  align?: 'left' | 'right';
}

export default function DropdownMenu({
  items,
  isOpen,
  onClose,
  triggerRef,
  align = 'right',
}: DropdownMenuProps) {
  const menuRef = useRef<HTMLDivElement>(null);
  const [position, setPosition] = useState({ top: 0, left: 0 });
  const [flipVertical, setFlipVertical] = useState(false);

  // Calculate position based on trigger element and available space
  useEffect(() => {
    if (isOpen && triggerRef.current) {
      const rect = triggerRef.current.getBoundingClientRect();
      const menuWidth = 192; // w-48 = 12rem = 192px
      const menuHeight = items.length * 40 + 12; // Approximate: 40px per item + padding
      const gap = 8; // mt-2 = 8px

      const viewportHeight = window.innerHeight;
      const viewportWidth = window.innerWidth;

      // Check if menu would overflow bottom of viewport
      const spaceBelow = viewportHeight - rect.bottom - gap;
      const spaceAbove = rect.top - gap;
      const shouldFlip = spaceBelow < menuHeight && spaceAbove > spaceBelow;

      setFlipVertical(shouldFlip);

      // Calculate vertical position
      const top = shouldFlip
        ? rect.top - menuHeight - gap
        : rect.bottom + gap;

      // Calculate horizontal position, ensuring it stays within viewport
      let left = align === 'right' ? rect.right - menuWidth : rect.left;
      // Clamp to viewport bounds with some padding
      left = Math.max(8, Math.min(left, viewportWidth - menuWidth - 8));

      setPosition({ top, left });
    }
  }, [isOpen, triggerRef, align, items.length]);

  // Close on click outside
  useEffect(() => {
    if (!isOpen) return;

    const handleClickOutside = (event: MouseEvent) => {
      if (
        menuRef.current &&
        !menuRef.current.contains(event.target as Node) &&
        triggerRef.current &&
        !triggerRef.current.contains(event.target as Node)
      ) {
        onClose();
      }
    };

    const handleEscape = (event: KeyboardEvent) => {
      if (event.key === 'Escape') {
        onClose();
      }
    };

    document.addEventListener('mousedown', handleClickOutside);
    document.addEventListener('keydown', handleEscape);
    return () => {
      document.removeEventListener('mousedown', handleClickOutside);
      document.removeEventListener('keydown', handleEscape);
    };
  }, [isOpen, onClose, triggerRef]);

  if (!isOpen) return null;

  // Find danger items to add separator before them
  const hasRegularItems = items.some(item => item.variant !== 'danger');
  const hasDangerItems = items.some(item => item.variant === 'danger');
  const needsSeparator = hasRegularItems && hasDangerItems;

  return createPortal(
    <div
      ref={menuRef}
      className={`fixed w-48 bg-white/95 dark:bg-gray-800/95 backdrop-blur-sm rounded-xl shadow-xl ring-1 ring-black/5 dark:ring-white/10 py-1.5 z-[9999] ${
        flipVertical ? 'origin-bottom animate-dropdown-in-up' : 'origin-top animate-dropdown-in'
      }`}
      style={{
        top: position.top,
        left: position.left,
      }}
    >
      {items.map((item, index) => {
        const isFirstDanger = item.variant === 'danger' &&
          (index === 0 || items[index - 1].variant !== 'danger');
        const showSeparator = needsSeparator && isFirstDanger;

        return (
          <div key={item.label}>
            {showSeparator && (
              <div className="border-t border-gray-200/60 dark:border-gray-600/60 my-1.5 mx-3" />
            )}
            <button
              onClick={() => {
                item.onClick();
                onClose();
              }}
              className={`w-full px-3 py-2 mx-1.5 text-left text-sm rounded-lg transition-colors flex items-center gap-2.5 ${
                item.variant === 'danger'
                  ? 'text-red-600 dark:text-red-400 hover:bg-red-50 dark:hover:bg-red-900/30'
                  : 'text-gray-700 dark:text-gray-200 hover:bg-gray-100 dark:hover:bg-gray-700/70'
              }`}
              style={{ width: 'calc(100% - 12px)' }}
            >
              {item.icon && (
                <span className={item.variant === 'danger' ? '' : 'text-gray-400 dark:text-gray-500'}>
                  {item.icon}
                </span>
              )}
              {item.label}
            </button>
          </div>
        );
      })}
    </div>,
    document.body
  );
}

// Common icons for menu items
export const MenuIcons = {
  play: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M14.752 11.168l-3.197-2.132A1 1 0 0010 9.87v4.263a1 1 0 001.555.832l3.197-2.132a1 1 0 000-1.664z" />
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M21 12a9 9 0 11-18 0 9 9 0 0118 0z" />
    </svg>
  ),
  trash: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M19 7l-.867 12.142A2 2 0 0116.138 21H7.862a2 2 0 01-1.995-1.858L5 7m5 4v6m4-6v6m1-10V4a1 1 0 00-1-1h-4a1 1 0 00-1 1v3M4 7h16" />
    </svg>
  ),
  branch: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M8 7h12m0 0l-4-4m4 4l-4 4m0 6H4m0 0l4 4m-4-4l4-4" />
    </svg>
  ),
  edit: (
    <svg className="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
      <path strokeLinecap="round" strokeLinejoin="round" strokeWidth={1.5} d="M11 5H6a2 2 0 00-2 2v11a2 2 0 002 2h11a2 2 0 002-2v-5m-1.414-9.414a2 2 0 112.828 2.828L11.828 15H9v-2.828l8.586-8.586z" />
    </svg>
  ),
};
