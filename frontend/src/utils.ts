import {MoonrakerPrinterState} from "@/api";

export function getPrinterStateKeyByValue(value: MoonrakerPrinterState): string | undefined {
    for (const key in MoonrakerPrinterState) {
        if (MoonrakerPrinterState[key as keyof typeof MoonrakerPrinterState] === value) {
            return key;
        }
    }
    return undefined;
}

export type PrinterStateInfo = {
    stateColor: string;
    stateText: string;
    isInError: boolean;
    isDisconnected: boolean;
}

export function getPrinterStateInfo(state: MoonrakerPrinterState): PrinterStateInfo {
    let stateText = getPrinterStateKeyByValue(state) ?? "Unknown";
    let isInError: boolean = false;
    let stateColor: string;

    switch (state) {
        case MoonrakerPrinterState.Ready:
            stateColor = "dark";
            break
        case MoonrakerPrinterState.Printing:
            stateColor = "primary";
            break;
        case MoonrakerPrinterState.Pause:
            stateColor = "warning";
            break;
        case MoonrakerPrinterState.KlippyShutdown:
        case MoonrakerPrinterState.KlippyError:
        case MoonrakerPrinterState.KlippyDisconnected:
        case MoonrakerPrinterState.Error:
        case MoonrakerPrinterState.InternalError:
            stateColor = "danger";
            isInError = true;
            break;
        case MoonrakerPrinterState.PrePrint:
        case MoonrakerPrinterState.KlippyStartup:
            stateColor = "info";
            break
        case MoonrakerPrinterState.Disconnected:
        default:
            stateColor = "secondary";
    }

    return {
        stateColor,
        stateText,
        isInError,
        isDisconnected: (state === MoonrakerPrinterState.Disconnected),
    }
}

export function secondsToDurationString(sec: number): string {
    if (Number.isNaN(sec)) return "NaN";
    if (sec < 0 || !Number.isFinite(sec)) return "N/A";

    if (sec < 3600) {
        return new Date(sec * 1000).toISOString().substring(14, 19)
    }

    return new Date(sec * 1000).toISOString().substring(11, 19)
}

export function getJobStatsColor(jobStatus: string): string {
    switch (jobStatus) {
        case "in_progress":
            return "primary";
        case "completed":
            return "success";
        case "cancelled":
            return "warning";
        case "interrupted":
        case "error":
        case "klippy_disconnect":
        case "klippy_shutdown":
        case "server_exit":
            return "danger";
        default:
            return "secondary";
    }
}
