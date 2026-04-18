# Razer Huntsman Matrix Map

Matrix dimensions: **6 rows × 22 cols** (132 positions total).
Orientation: standard desk-facing. On the wall (vertical, left side down), rows run bottom→top.

Legend:
- **Bold** = confirmed via code or live testing
- `?` = unknown / not yet mapped
- _italic_ = guessed, needs verification
- `—` = confirmed empty (matrix position exists but no physical key)

## Grid

| row \ col | 0 | 1 | 2 | 3 | 4 | 5 | 6 | 7 | 8 | 9 | 10 | 11 | 12 | 13 | 14 | 15 | 16 | 17 | 18 | 19 | 20 | 21 |
|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|---|
| **0** | ? | ? | ? | ? | ? | ? | ? | **F5** | **F6** | **F7** | **F8** | **F9** | **F10** | **F11** | **F12** | ? | ? | ? | ? | ? | ? | ? |
| **1** | ? | **`** | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | **PgUp** | **NumLk** | **/** | **\*** | _-_ |
| **2** | ? | **Tab** | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | **PgDn** | ? | **8** | ? | _+_ |
| **3** | ? | **Caps** | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | **4** | **5** | **6** | ? |
| **4** | ? | **LSh** | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? |
| **5** | ? | **LCtrl** | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? | ? |

## Status

**Confirmed (22 positions):**
- Row 0: F5-F12 consecutive at cols 7-14 (no gap)
- Col 1 (left edge): `` ` ``, Tab, Caps, LShift, LCtrl at rows 1-5
- Numpad: NumLk/`/`/`*` (row 1, cols 18-20), 8 (row 2, col 19), 4/5/6 (row 3, cols 18-20)
- Nav cluster: PgUp {1,17}, PgDn {2,17}

**Approximate (needs verification):**
- Numpad `-` {1,21}, Numpad `+` {2,21}

**Not yet mapped:**
- F1-F4 (row 0, presumably cols 3-6 or similar)
- ESC (row 0, probably col 0 or 1)
- Main alpha block (rows 1-4, cols 2-14)
- Bottom row modifiers and space (row 5)
- Nav cluster remainder: Ins, Home, Del, End, arrow keys
- Remaining numpad: 7/9, 1/2/3, 0, `.`, Enter

## How to fill this in

Run the discover tool on the Pi:

```bash
ssh pi@raspberrypi.local /opt/huntsman-panel/discover
```

It lights one position at a time and waits for Enter. Watch which physical key lights up, update this map, then commit.

Cross-reference with the evdev code in `cmd/daemon/main.go` when binding macros — the matrix position (for lighting) and the evdev key code (for input) are independent.
