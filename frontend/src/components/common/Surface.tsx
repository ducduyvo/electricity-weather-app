import Paper from "@mui/material/Paper";
import type { ReactNode } from "react";

export function Panel({
  children,
  className = "",
}: {
  children: ReactNode;
  className?: string;
}) {
  return (
    <Paper
      component="section"
      variant="outlined"
      className={`panel ${className}`}
    >
      {children}
    </Paper>
  );
}

export function StatCard({ children }: { children: ReactNode }) {
  return (
    <Paper component="article" variant="outlined" className="card">
      {children}
    </Paper>
  );
}
