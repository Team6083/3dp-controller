import {PrinterPrinterState as PrinterState} from "./api";

export function getPrinterStateKeyByValue(value: PrinterState): string | undefined {
    for (const key in PrinterState) {
        if (PrinterState[key as keyof typeof PrinterState] === value) {
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

export function getPrinterStateInfo(state: PrinterState): PrinterStateInfo {
    const stateText = getPrinterStateKeyByValue(state) ?? "Unknown";
    let isInError: boolean = false;
    let stateColor: string;

    switch (state) {
        case PrinterState.Ready:
            stateColor = "dark";
            break
        case PrinterState.Printing:
            stateColor = "primary";
            break;
        case PrinterState.Pause:
            stateColor = "warning";
            break;
        case PrinterState.Error:
        case PrinterState.InternalError:
            stateColor = "danger";
            isInError = true;
            break;
        case PrinterState.PrePrint:
            stateColor = "info";
            break
        case PrinterState.Disconnected:
        default:
            stateColor = "secondary";
    }

    return {
        stateColor,
        stateText,
        isInError,
        isDisconnected: (state === PrinterState.Disconnected),
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
