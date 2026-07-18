import {
    WebPrinter,
    PrinterErrorInfo,
    PrinterJob,
    PrinterPrinterState,
} from "./api";
import {getJobStatsColor, secondsToDurationString} from "./utils";

export const PrinterState = PrinterPrinterState;
export type PrinterState = PrinterPrinterState;

export interface Job {
    jobId: string;
    status: string;
    name: string;
    hasThumbnail: boolean;

    progress?: number;
    printDuration?: number;
    totalDuration?: number;
    estimatedRemainingSec?: number;
}

export function convertJob(job: PrinterJob): Job {
    return {
        jobId: job.job_id!,
        status: job.status!,
        name: job.name!,
        hasThumbnail: job.has_thumbnail ?? false,

        progress: job.progress,
        printDuration: job.print_duration,
        totalDuration: job.total_duration,
        estimatedRemainingSec: job.remaining_sec,
    }
}

export interface ErrorDetail {
    code?: number;
    message: string;
}

export function convertErrorDetail(errorDetail: PrinterErrorInfo): ErrorDetail {
    return {
        code: errorDetail.code,
        message: errorDetail.message!,
    }
}

export interface Printer {
    key: string;
    name: string;
    url: string;
    type: string;

    registeredJobId: string;
    allowNoRegisteredPrint: boolean;
    noPauseDuration: number;

    state: PrinterState;
    printerNotOpen: boolean;
    displayMessage?: string;
    errorMessage?: string;
    lastUpdateTime: Date;

    job?: Job;
}

export function convertPrinter(printer: WebPrinter): Printer {
    let displayMessage = printer.message;
    if (displayMessage?.trim() === "") displayMessage = undefined;

    const errorDetail = printer.error_detail ? convertErrorDetail(printer.error_detail) : undefined;

    return {
        key: printer.key!,
        name: printer.name!,
        url: printer.url!,
        type: printer.type!,

        registeredJobId: printer.registered_job_id ?? "",
        allowNoRegisteredPrint: printer.allow_no_register_print!,
        noPauseDuration: printer.no_pause_duration!,

        state: printer.state!,
        printerNotOpen: false,
        displayMessage,
        errorMessage: errorDetail?.message ?? printer.message,
        lastUpdateTime: new Date(printer.last_update_time!),

        job: printer.job ? convertJob(printer.job) : undefined,
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
    const job = printer.job;
    if (!job) return undefined;

    const jobInfo: ActiveJobInfo | JobInfo = {
        id: job.jobId,
        fileName: job.name,
        status: job.status,
        statusColor: getJobStatsColor(job.status),
        isActive: false,
        imageUrl: job.hasThumbnail ? `/printers/${printer.key}/latest_thumb` : undefined,
    };

    if (job.status === "in_progress") {
        const printTime = typeof job.printDuration === "number" ?
            secondsToDurationString(job.printDuration) : undefined;

        const totalTime = typeof job.totalDuration === "number" ?
            secondsToDurationString(job.totalDuration) : undefined;

        let estRemainSec = job.estimatedRemainingSec;
        if (typeof estRemainSec === "number" && estRemainSec < 0) {
            estRemainSec = 0;
        }

        const willPause = !printer.allowNoRegisteredPrint && job.jobId !== printer.registeredJobId;
        const pauseRemainSec = typeof job.printDuration === "number" ?
            Math.max(printer.noPauseDuration - job.printDuration, 0) : undefined;

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
