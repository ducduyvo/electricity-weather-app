import { createTheme } from "@mui/material/styles";

export const theme = createTheme({
  palette: {
    primary: { main: "#046ad1" },
    secondary: { main: "#ffc126" },
    background: { default: "#f9fafa", paper: "#ffffff" },
    text: { primary: "#373f46", secondary: "#505a63" },
    warning: { main: "#8e6e01" },
    error: { main: "#ba2225" },
  },
  typography: {
    fontFamily: 'Inter, ui-sans-serif, system-ui, -apple-system, BlinkMacSystemFont, "Segoe UI", sans-serif',
    fontSize: 14,
    h1: { fontSize: "2rem", fontWeight: 600 },
    h2: { fontSize: "1.125rem", fontWeight: 600 },
    h3: { fontSize: "0.875rem", fontWeight: 600 },
    button: { textTransform: "none", fontWeight: 600 },
  },
  shape: { borderRadius: 4 },
  components: {
    MuiButton: {
      defaultProps: { variant: "contained", disableElevation: true },
      styleOverrides: { root: { minHeight: 40, backgroundColor: "#222e46", "&:hover": { backgroundColor: "#4c5d80" } } },
    },
    MuiToggleButton: {
      styleOverrides: { root: { minWidth: 152, "&.Mui-selected": { backgroundColor: "#a3b1cc", color: "#222e46", "&:hover": { backgroundColor: "#a3b1cc" } } } },
    },
    MuiSelect: {
      defaultProps: { size: "small" },
      styleOverrides: {
        select: { paddingLeft: 12, paddingRight: "40px !important", backgroundColor: "#fff" },
        icon: { right: 12 },
      },
    },
    MuiPaper: { defaultProps: { elevation: 0 } },
    MuiTooltip: { defaultProps: { arrow: true } },
    MuiTableCell: { styleOverrides: { root: { whiteSpace: "nowrap", fontSize: 12, padding: "12px 16px", fontVariantNumeric: "tabular-nums" }, head: { fontWeight: 600, backgroundColor: "#eceff1" } } },
  },
});
