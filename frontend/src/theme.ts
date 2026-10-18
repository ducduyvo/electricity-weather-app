import { alpha, createTheme } from "@mui/material/styles";
import { colors, cssTokens, fontFamily, borderRadius } from "./designTokens";

export const theme = createTheme({
  palette: {
    primary: { main: colors.blue },
    secondary: { main: colors.yellow },
    background: { default: colors.background, paper: colors.surface },
    text: { primary: colors.text, secondary: colors.textSecondary },
    warning: { main: colors.warningText },
    error: { main: colors.errorText },
  },
  typography: {
    fontFamily,
    fontSize: 14,
    h1: { fontSize: "2rem", fontWeight: 600 },
    h2: { fontSize: "1.125rem", fontWeight: 600 },
    h3: { fontSize: "0.875rem", fontWeight: 600 },
    button: { textTransform: "none", fontWeight: 600 },
  },
  shape: { borderRadius },
  components: {
    MuiCssBaseline: { styleOverrides: { ":root": cssTokens } },
    MuiButton: {
      defaultProps: { variant: "contained", disableElevation: true },
      styleOverrides: {
        root: {
          minHeight: 40,
          backgroundColor: colors.navy,
          "&:hover": { backgroundColor: colors.navyHover },
        },
      },
    },
    MuiToggleButton: {
      styleOverrides: {
        root: {
          minWidth: 152,
          "&.Mui-selected": {
            backgroundColor: colors.accent,
            color: colors.navy,
            "&:hover": { backgroundColor: colors.accent },
          },
        },
      },
    },
    MuiSelect: {
      defaultProps: { size: "small" },
      styleOverrides: {
        select: {
          paddingLeft: 12,
          paddingRight: "40px !important",
          backgroundColor: colors.surface,
        },
        icon: { right: 12 },
      },
    },
    MuiOutlinedInput: {
      styleOverrides: {
        root: {
          // Menu dismissal restores focus; retain strong styling for keyboard
          // focus and open menus, but not restored mouse focus on a closed select.
          '&.Mui-focused:has(.MuiSelect-select):not(:has(.MuiSelect-select:focus-visible)):not(:has(.MuiSelect-select[aria-expanded="true"])) .MuiOutlinedInput-notchedOutline':
            {
              borderColor: alpha(colors.text, 0.23),
              borderWidth: 1,
            },
        },
      },
    },
    MuiPaper: { defaultProps: { elevation: 0 } },
    MuiTooltip: { defaultProps: { arrow: true } },
    MuiTableCell: {
      styleOverrides: {
        root: {
          whiteSpace: "nowrap",
          fontSize: 12,
          padding: "12px 16px",
          fontVariantNumeric: "tabular-nums",
        },
        head: { fontWeight: 600, backgroundColor: colors.tableHeader },
      },
    },
  },
});
