import {
    MoonrakerGCodeMetadata,
    MoonrakerJob,
    MoonrakerPrinterObjectPrintStats,
    MoonrakerPrinterObjectVirtualSDCard,
    MoonrakerPrinterState,
    WebPrinter
} from "@/api";
import {getJobStatsColor, secondsToDurationString} from "@/utils";
import {max} from "@popperjs/core/lib/utils/math";


export interface GCodeMetadata {
    fileName: string;
    estimatedTime?: number;
    uuid: string;
    hasThumb: boolean;
}

export function convertGCodeMetadata(gcodeMeta: MoonrakerGCodeMetadata): GCodeMetadata {
    return {
        fileName: gcodeMeta.filename!,
        estimatedTime: gcodeMeta.estimated_time,
        uuid: gcodeMeta.uuid!,
        hasThumb: (gcodeMeta.thumbnails && gcodeMeta.thumbnails.length > 0) ?? false
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
    lastUpdateTime: Date;

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
        lastUpdateTime: new Date(printer.last_update_time!),

        printerStats: printer.printer_stats ? convertPrinterStats(printer.printer_stats) : undefined,
        virtualSD: printer.virtual_sd_card ? convertVirtualSD(printer.virtual_sd_card) : undefined,

        loadedFile: printer.loaded_file ? convertGCodeMetadata(printer.loaded_file) : undefined,
        latestJob: printer.latest_job ? convertJob(printer.latest_job) : undefined
    }
}

export type JobInfo = {
    id: string;
    owner?: string;
    isActive: false;

    status: string;
    statusColor: string;

    fileName: string;
    imageUrl?: string;
}

export type ActiveJobInfo = Omit<JobInfo, 'isActive'> & {
    isActive: true;

    jobWillPause: boolean;
    pauseRemainSec?: number;

    estRemainSec?: number;
    printTime?: string;
    totalTime?: string;
}

export function getLatestJobInfo(printer: Printer): ActiveJobInfo | JobInfo | undefined {
    if (!printer.latestJob) return undefined;

    let jobInfo: ActiveJobInfo | JobInfo = {
        id: printer.latestJob.jobId,
        fileName: printer.latestJob.filename,
        status: printer.latestJob.status,
        statusColor: getJobStatsColor(printer.latestJob.status),
        isActive: false,
        imageUrl: printer.latestJob.metadata?.hasThumb ? `/printers/${printer.key}/latest_thumb` : undefined,
    };

    if (printer.loadedFile?.uuid === printer.latestJob.metadata?.uuid) {
        const printTime = printer.printerStats ?
            secondsToDurationString(printer.printerStats.printDuration) : undefined;

        const totalTime = printer.printerStats ?
            secondsToDurationString(printer.printerStats.totalDuration) : undefined;

        let estRemainSec: number | undefined;
        if (printer.virtualSD) {
            if (typeof printer.loadedFile?.estimatedTime === "number") {
                const estimatedTime = printer.loadedFile.estimatedTime;
                const progressTime = printer.virtualSD.progress * estimatedTime;
                estRemainSec = estimatedTime - progressTime;
            } else if (printer.printerStats && printer.virtualSD.progress > 0) {
                const printDuration = printer.printerStats.printDuration;
                let totalTime = printDuration / printer.virtualSD.progress;
                estRemainSec = totalTime - printDuration;
            }
        }

        if (typeof estRemainSec === "number" && estRemainSec < 0) {
            estRemainSec = 0;
        }

        const willPause = !printer.allowNoRegisteredPrint && jobInfo.id !== printer.registeredJobId;
        const pauseRemainSec = printer.printerStats ?
            max(printer.noPauseDuration - printer.printerStats.printDuration, 0) : undefined;

        return {
            ...jobInfo,
            isActive: true,
            estRemainSec,
            printTime,
            totalTime,
            jobWillPause: willPause,
            pauseRemainSec,
        }
    }

    return jobInfo;
}
