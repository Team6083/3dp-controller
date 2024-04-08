'use client'

import Card from "react-bootstrap/Card";
import {Printer} from "@/types";
import {moonraker_PrinterState} from "@/api";
import Badge from "react-bootstrap/Badge";

function getKeyByValue(value: moonraker_PrinterState): string | undefined {
    for (const key in moonraker_PrinterState) {
        if (moonraker_PrinterState[key as keyof typeof moonraker_PrinterState] === value) {
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

    console.log(sec);

    return new Date(sec * 1000).toISOString().substring(11, 19)
}

type PrinterCardProps = {
    printer: Printer
}

function PrinterCard({printer}: PrinterCardProps) {
    let stateText = getKeyByValue(printer.state) ?? "Unknown";
    let isPrinterInErrorState: boolean = false;
    let stateColor: string;

    switch (printer.state) {
        case moonraker_PrinterState.Ready:
            stateColor = "primary";
            if (printer.printerStats?.state === "complete") stateText = "Complete";
            else if (printer.printerStats?.state === "cancelled") stateText = "Cancelled";
            break
        case moonraker_PrinterState.Printing:
            stateColor = "success";
            break;
        case moonraker_PrinterState.Pause:
            stateColor = "warning";
            break;
        case moonraker_PrinterState.KlippyShutdown:
        case moonraker_PrinterState.KlippyError:
        case moonraker_PrinterState.KlippyDisconnected:
        case moonraker_PrinterState.Error:
        case moonraker_PrinterState.InternalError:
            stateColor = "danger";
            isPrinterInErrorState = true;
            break;
        case moonraker_PrinterState.PrePrint:
        case moonraker_PrinterState.KlippyStartup:
            stateColor = "info";
            break
        case moonraker_PrinterState.Disconnected:
        default:
            stateColor = "secondary";
    }

    const printerDuration = printer.printerStats ?
        secondsToDurationString(printer.printerStats.printDuration) : undefined;

    const totalDuration = printer.printerStats ?
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

    const eta = typeof estRemainSec === "number" ? new Date(Date.now() + estRemainSec * 1000) : undefined;

    return <Card border={stateColor}>
        <Card.Header>
            {printer.name} - <span className={`text-${stateColor} fw-semibold`}>{stateText}</span>
        </Card.Header>
        <Card.Body>
            {isPrinterInErrorState ? <>
                <Card.Title>Error</Card.Title>
                <Card.Text>{printer.errorMessage}</Card.Text>
            </> : <>
                {printer.loadedFile ? <>
                    <Card.Title>Current Job: {printer.loadedFile.fileName}</Card.Title>
                    {estRemainSec ? <Card.Subtitle className="mb-2 text-muted">
                        Estimate: {secondsToDurationString(estRemainSec)}, ETA: {eta?.toLocaleTimeString()}
                    </Card.Subtitle> : null}

                    <Card.Text>Time: {printerDuration} / {totalDuration}</Card.Text>
                </> : null}

                {printer.latestJob ? <>
                    <Card.Text>Job Id: <Badge>{printer.latestJob?.jobId}</Badge> Job owner: N/A</Card.Text>
                </> : null}
            </>}
        </Card.Body>
    </Card>
}

export default PrinterCard
