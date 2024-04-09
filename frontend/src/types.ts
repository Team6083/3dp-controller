import {
    MoonrakerGCodeMetadata,
    MoonrakerJob,
    MoonrakerPrinterObjectPrintStats,
    MoonrakerPrinterObjectVirtualSDCard,
    MoonrakerPrinterState,
    WebPrinter
} from "@/api";


export interface GCodeMetadata {
    fileName: string;
    estimatedTime?: number;
    uuid: string;
}

export function convertGCodeMetadata(gcodeMeta: MoonrakerGCodeMetadata): GCodeMetadata {
    return {
        fileName: gcodeMeta.filename!,
        estimatedTime: gcodeMeta.estimated_time,
        uuid: gcodeMeta.uuid!,
    }
}

export interface Job {
    jobId: string;
    status: string;
    filename: string;
    metadata?: GCodeMetadata;
}

export function convertJob(job: MoonrakerJob): Job {
    return {
        jobId: job.job_id!,
        status: job.status!,
        filename: job.filename!,
        metadata: job.metadata ? convertGCodeMetadata(job.metadata) : undefined,
    }
}

export interface PrinterStats {
    printDuration: number;
    totalDuration: number;
    filamentUsed: number;
    state: string;
}

export function convertPrinterStats(printerStats: MoonrakerPrinterObjectPrintStats): PrinterStats {
    return {
        printDuration: printerStats.print_duration!,
        totalDuration: printerStats.total_duration!,
        filamentUsed: printerStats.filament_used!,
        state: printerStats.state!,
    }
}

export interface VirtualSD {
    progress: number;
    isActive: boolean;
}

export function convertVirtualSD(virtualSD: MoonrakerPrinterObjectVirtualSDCard): VirtualSD {
    return {
        progress: virtualSD.progress!,
        isActive: virtualSD.is_active!,
    }
}

export interface Printer {
    key: string;
    name: string;
    url: string;

    registeredJobId: string;
    allowNoRegisteredPrint: boolean;
    noPauseDuration: number;

    state: MoonrakerPrinterState;
    printerNotOpen: boolean;
    displayMessage?: string;
    errorMessage?: string;

    printerStats?: PrinterStats;
    virtualSD?: VirtualSD;

    loadedFile?: GCodeMetadata;
    latestJob?: Job;
}

export function convertPrinter(printer: WebPrinter): Printer {
    let displayMessage = printer.display_status?.message;
    if (displayMessage?.trim() === "") displayMessage = undefined;

    return {
        key: printer.key!,
        name: printer.name!,
        url: printer.url!,

        registeredJobId: printer.registered_job_id ?? "",
        allowNoRegisteredPrint: printer.allow_no_register_print!,
        noPauseDuration: printer.no_pause_duration!,

        state: printer.state!,
        printerNotOpen: false,
        displayMessage,
        errorMessage: printer.message,

        printerStats: printer.printer_stats ? convertPrinterStats(printer.printer_stats) : undefined,
        virtualSD: printer.virtual_sd_card ? convertVirtualSD(printer.virtual_sd_card) : undefined,

        loadedFile: printer.loaded_file ? convertGCodeMetadata(printer.loaded_file) : undefined,
        latestJob: printer.latest_job ? convertJob(printer.latest_job) : undefined
    }
}
