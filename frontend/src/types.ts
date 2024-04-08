import {
    moonraker_GCodeMetadata,
    moonraker_Job,
    moonraker_PrinterObjectPrintStats,
    moonraker_PrinterObjectVirtualSDCard,
    moonraker_PrinterState,
    web_Printer
} from "@/api";

export interface GCodeMetadata {
    fileName: string;
    estimatedTime?: number;
}

export function convertGCodeMetadata(gcodeMeta: moonraker_GCodeMetadata): GCodeMetadata {
    return {
        fileName: gcodeMeta.filename!,
        estimatedTime: gcodeMeta.estimated_time,
    }
}

export interface Job {
    jobId: string;
    status: string;
    filename: string;
    metadata?: GCodeMetadata;
}

export function convertJob(job: moonraker_Job): Job {
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
    state: string;
}

export function convertPrinterStats(printerStats: moonraker_PrinterObjectPrintStats): PrinterStats {
    return {
        printDuration: printerStats.print_duration!,
        totalDuration: printerStats.total_duration!,
        state: printerStats.state!,
    }
}

export interface VirtualSD {
    progress: number;
    isActive: boolean;
}

export function convertVirtualSD(virtualSD: moonraker_PrinterObjectVirtualSDCard): VirtualSD {
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

    state: moonraker_PrinterState;
    errorMessage?: string;

    printerStats?: PrinterStats;
    virtualSD?: VirtualSD;

    loadedFile?: GCodeMetadata;
    latestJob?: Job;
}

export function convertPrinter(printer: web_Printer): Printer {
    return {
        key: printer.key!,
        name: printer.name!,
        url: printer.url!,

        registeredJobId: printer.registered_job_id ?? "",
        allowNoRegisteredPrint: printer.allow_no_register_print!,

        state: printer.state!,
        errorMessage: printer.message,

        printerStats: printer.printer_stats ? convertPrinterStats(printer.printer_stats) : undefined,
        virtualSD: printer.virtual_sd_card ? convertVirtualSD(printer.virtual_sd_card) : undefined,

        loadedFile: printer.loaded_file ? convertGCodeMetadata(printer.loaded_file) : undefined,
        latestJob: printer.latest_job ? convertJob(printer.latest_job) : undefined
    }
}
