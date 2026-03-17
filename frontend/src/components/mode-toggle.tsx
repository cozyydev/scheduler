
import { Button } from "@/components/ui/button";
import { useTheme } from "@/components/theme-provider";
import { ContrastIcon } from "./contrast-icon";

export function ModeToggle() {
  const { setTheme, theme } = useTheme();

  const current = theme; // handles "system"
  const toggleTheme = () => {
    const next = current === "dark" ? "light" : "dark";
    setTheme(next);
  };

  return (
    <div className="fixed top-2 right-2">
      <Button
        variant="ghost"
        size="icon"
        onClick={toggleTheme}
        aria-label={
          current === "dark" ? "Switch to light mode" : "Switch to dark mode"
        }
        className={`h-5 w-5 rounded-full transition-colors ${current === "dark"
            ? "ring-primary bg-muted text-primary ring-2"
            : "ring-primary bg-muted text-primary ring-2"
          }`}
      >
        <ContrastIcon className="h-3 w-3" />
        <span className="sr-only">{current === "dark" ? "Light" : "Dark"}</span>
      </Button>
    </div>
  );
}
