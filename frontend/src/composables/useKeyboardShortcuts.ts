import { onMounted, onBeforeUnmount } from 'vue'

export function useKeyboardShortcuts(handlers: {
  onSelectAll?: (event: KeyboardEvent) => void
  onDelete?: (event: KeyboardEvent) => void
  onNavigatePrev?: (event: KeyboardEvent) => void
  onNavigateNext?: (event: KeyboardEvent) => void
  onEscape?: (event: KeyboardEvent) => void
}) {
  function handleKeydown(event: KeyboardEvent) {
    // Ignore if user is typing in an input
    const target = event.target as HTMLElement
    if (target && ['INPUT', 'TEXTAREA', 'SELECT'].includes(target.tagName)) {
      return
    }

    // Ctrl/Cmd + A - Select all
    if ((event.ctrlKey || event.metaKey) && event.key === 'a') {
      event.preventDefault()
      handlers.onSelectAll?.(event)
    }

    // Delete - Delete selected
    else if (event.key === 'Delete') {
      handlers.onDelete?.(event)
    }

    // Arrow keys - Navigation
    else if (event.key === 'ArrowLeft') {
      handlers.onNavigatePrev?.(event)
    }
    else if (event.key === 'ArrowRight') {
      handlers.onNavigateNext?.(event)
    }

    // Escape - Close/cancel
    else if (event.key === 'Escape') {
      handlers.onEscape?.(event)
    }
  }

  onMounted(() => {
    document.addEventListener('keydown', handleKeydown)
  })

  onBeforeUnmount(() => {
    document.removeEventListener('keydown', handleKeydown)
  })

  return {
    destroy: () => {
      document.removeEventListener('keydown', handleKeydown)
    }
  }
}
