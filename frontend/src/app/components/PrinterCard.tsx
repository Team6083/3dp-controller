'use client'

import Badge from "react-bootstrap/Badge";
import Button from "react-bootstrap/Button"
import Card from "react-bootstrap/Card";

import {Printer} from "@/types";
import {MoonrakerPrinterState} from "@/api";

function getKeyByValue(value: MoonrakerPrinterState): string | undefined {
    for (const key in MoonrakerPrinterState) {
        if (MoonrakerPrinterState[key as keyof typeof MoonrakerPrinterState] === value) {
            return key;
        }
    }
    return undefined;
}

function secondsToDurationString(sec: number): string {
    if (Number.isNaN(sec)) return "NaN";
    if (sec < 0 || !Number.isFinite(sec)) return "N/A";

    if (sec < 3600) {
        return new Date(sec * 1000).toISOString().substring(14, 19)
    }

    return new Date(sec * 1000).toISOString().substring(11, 19)
}

type PrinterCardProps = {
    printer: Printer
}

type JobInfo = {
    id: string;
    fileName: string;
    status: string;
    statusColor: string;
    owner?: string;
    isActive: false;
}

type ActiveJobInfo = Omit<JobInfo, 'isActive'> & {
    isActive: true;
    estRemainSec?: number;
    printTime?: string;
    totalTime?: string;
}

function PrinterCard({printer}: PrinterCardProps) {
    let stateText = getKeyByValue(printer.state) ?? "Unknown";
    let isPrinterInErrorState: boolean = false;
    let isPrinterDisconnected: boolean = printer.state === MoonrakerPrinterState.Disconnected;
    let stateColor: string;

    switch (printer.state) {
        case MoonrakerPrinterState.Ready:
            stateColor = "dark";

            if (printer.printerStats?.state === "complete") {
                stateText = "Complete";
                stateColor = "success";
            } else if (printer.printerStats?.state === "cancelled") {
                stateText = "Cancelled";
            }
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
            isPrinterInErrorState = true;
            break;
        case MoonrakerPrinterState.PrePrint:
        case MoonrakerPrinterState.KlippyStartup:
            stateColor = "info";
            break
        case MoonrakerPrinterState.Disconnected:
        default:
            stateColor = "secondary";
    }

    const sdPercent = printer.virtualSD && printer.virtualSD.isActive ?
        (printer.virtualSD.progress * 100).toFixed(1) + "%" : undefined;

    let jobInfo: ActiveJobInfo | JobInfo | undefined;
    if (printer.latestJob) {
        let jobStatusColor: string;
        switch (printer.latestJob.status) {
            case "in_progress":
                jobStatusColor = "primary";
                break;
            case "completed":
                jobStatusColor = "success";
                break;
            case "cancelled":
                jobStatusColor = "warning";
                break;
            case "interrupted":
            case "error":
            case "klippy_disconnect":
            case "klippy_shutdown":
            case "server_exit":
                jobStatusColor = "danger";
                break;
            default:
                jobStatusColor = "secondary";
        }

        jobInfo = {
            id: printer.latestJob.jobId,
            status: printer.latestJob.status,
            statusColor: jobStatusColor,
            fileName: printer.latestJob.filename,
            isActive: false,
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

            jobInfo = {
                ...jobInfo,
                isActive: true,
                estRemainSec,
                printTime,
                totalTime
            }
        }
    }

    const printerWillPause: boolean = (!printer.allowNoRegisteredPrint &&
        jobInfo?.isActive && jobInfo.id !== printer.registeredJobId) ?? false;
    let pauseRemainSec = printer.printerStats ?
        printer.noPauseDuration - printer.printerStats.printDuration : undefined;
    if (typeof pauseRemainSec === "number" && pauseRemainSec < 0) pauseRemainSec = 0;

    const printerNotOpen = printer.printerNotOpen;
    const bgColor = printerWillPause ? "warning" : (printerNotOpen ? "danger" : undefined);
    const stateTextColor = bgColor === "warning" ? "dark" :
        (bgColor === "danger" ? "light" : undefined);

    return <Card
        border={bgColor ? undefined : stateColor}
        bg={bgColor}
        text={bgColor === "danger" ? "light" : undefined}
    >
        <Card.Header>
            {printer.name} -{" "}
            <span className={`text-${stateTextColor} fw-semibold`}>
                {stateText} {sdPercent ?? null}
            </span>
        </Card.Header>
        <Card.Body>
            {isPrinterDisconnected ? <>

            </> : (
                isPrinterInErrorState ? <>
                    <Card.Title>Error</Card.Title>
                    <Card.Text>{printer.errorMessage}</Card.Text>
                </> : <>
                    {jobInfo ? <>
                        <Card.Title>
                            {jobInfo.isActive ? "Current" : "Latest"} Job:{" "}
                            <Badge bg={jobInfo.statusColor}>{jobInfo.id}</Badge>
                        </Card.Title>
                        <Card.Subtitle className="mb-2 text-muted text-truncate">{jobInfo.fileName}</Card.Subtitle>

                        {jobInfo.isActive && jobInfo.estRemainSec ?
                            <Card.Text className="mb-0">
                                Estimate: {secondsToDurationString(jobInfo.estRemainSec)},
                                ETA: {new Date(Date.now() + jobInfo.estRemainSec * 1000).toLocaleTimeString()}
                            </Card.Text> : null}

                        {jobInfo.isActive ? <Card.Text>
                            <abbr title="Print / Total">Time</abbr>: {jobInfo.printTime} / {jobInfo.totalTime}
                        </Card.Text> : null}
                    </> : null}

                    {printerWillPause ? <>
                        <Card.Subtitle>
                            Job will be paused
                            {pauseRemainSec ?
                                ` after ${secondsToDurationString(pauseRemainSec)}` : null
                            }, please register.
                        </Card.Subtitle>

                        <div className="d-grid gap-2 mt-3">
                            <Button variant="success">Register Job</Button>
                        </div>
                    </> : null}
                </>
            )}
        </Card.Body>
    </Card>
}

export default PrinterCard
