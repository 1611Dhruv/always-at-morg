# Game Renderer

This directory contains game rendering logic (converting game state to ASCII art).

## Purpose

- Render game world as ASCII art
- Render players, entities, items
- Handle camera/viewport logic
- Convert game coordinates to screen coordinates

## Integration Point

The renderer is called from `ui/screen_main.go`:

```go
func (m Model) renderGamePanel(width, height int) string {
    // Get game state from connection manager
    gameState := m.connMgr.GetState()

    // Render the game world
    return renderer.RenderWorld(gameState, width, height)
}
```

## TODO

Your teammate's termloop rendering can be refactored here.

**Suggested files:**
- `world.go` - Render game world
- `entities.go` - Render players, items, etc.
- `viewport.go` - Camera and viewport logic
- `ascii.go` - ASCII art helpers
