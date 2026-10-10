import IconButton from "@mui/material/IconButton";
import Tooltip from "@mui/material/Tooltip";
import { useI18n } from "../../i18n";

export function InfoTooltip({ text }: { text: string }) {
  const { t } = useI18n();
  return (
    <Tooltip title={text}>
      <IconButton size="small" aria-label={t("weather.more_information")} className="info-tooltip-trigger">?</IconButton>
    </Tooltip>
  );
}
