import {useMemo} from "react";

import Badge from "react-bootstrap/Badge";
import Button from "react-bootstrap/Button"
import Card from "react-bootstrap/Card";

import {getLatestJobInfo, Printer, PrinterState} from "../../types";
import {getPrinterStateInfo, secondsToDurationString} from "../../utils";

type PrinterCardProps = {
    printer: Printer
    apiURLBase: string
}

function PrinterCard({printer, apiURLBase}: PrinterCardProps) {
    const {
        stateText,
        stateColor,
        isPrinterDisconnected,
        isPrinterInErrorState,
    } = useMemo(() => {
        const info = getPrinterStateInfo(printer.state);
        let {stateText, stateColor} = info;

        if (printer.state === PrinterState.Ready) {
            if (printer.job?.status === "completed") {
                stateText = "Complete";
                stateColor = "success";
            } else if (printer.job?.status === "cancelled") {
                stateText = "Cancelled";
            }
        }

        return {
            stateText,
            stateColor,
            isPrinterDisconnected: info.isDisconnected,
            isPrinterInErrorState: info.isInError,
        }
    }, [printer.state, printer.job?.status]);

    const sdPercent = typeof printer.job?.progress === "number" ?
        (printer.job.progress * 100).toFixed(1) + "%" : undefined;

    const jobInfo = useMemo(() => {
        return getLatestJobInfo(printer);
    }, [printer]);


    const printerNotOpen = printer.printerNotOpen;

    const bgColor = (jobInfo?.isActive && jobInfo.jobWillPause) ?
        "warning" : (printerNotOpen ? "danger" : undefined);
    const hasBg = bgColor !== undefined;
    const isDarkBg = bgColor === "danger";

    const stateTextColor = !hasBg ? stateColor : (isDarkBg ? "light" : "dark");

    return <Card
        border={bgColor ? undefined : stateColor}
        bg={bgColor}
        text={isDarkBg ? "light" : undefined}
        className="shadow"
    >
        <Card.Header>
            {printer.name} -{" "}
            <span className={`text-${stateTextColor} fw-semibold`}>
                {stateText} {sdPercent ?? null}
            </span>
        </Card.Header>
        {jobInfo?.imageUrl ?
            <div className="overflow-hidden align-content-center text-center" style={{height: "150px"}}>
                <Card.Img className="thumb-img" src={apiURLBase + jobInfo.imageUrl}/>
            </div> : null}
        <Card.Body>
            {(() => {
                if (isPrinterDisconnected) {
                    return <></>
                }

                if (printer.printerNotOpen) {
                    return <>
                        <Card.Title>This printer is not open for use</Card.Title>
                        <Card.Subtitle>Please contact admin</Card.Subtitle>

                        <div className="d-grid gap-2 mt-3">
                            <Button>Admin Unlock</Button>
                        </div>
                    </>
                }

                if (isPrinterInErrorState) {
                    return <>
                        <Card.Title>Error</Card.Title>
                        <Card.Text>{printer.errorMessage}</Card.Text>
                    </>
                }

                return <>
                    {jobInfo ? <>
                        <Card.Title>
                            {jobInfo.isActive ? "Current" : "Latest"} Job:{" "}
                            <Badge bg={jobInfo.statusColor}>{jobInfo.id}</Badge>
                        </Card.Title>
                        <Card.Subtitle
                            className={`mb-2 ${isDarkBg ? "" : "text-muted"} text-truncate`}>{jobInfo.fileName}</Card.Subtitle>

                        {jobInfo.isActive && jobInfo.estRemainSec ?
                            <Card.Text className="mb-0">
                                Estimate: {secondsToDurationString(jobInfo.estRemainSec)},
                                ETA: {new Date(Date.now() + jobInfo.estRemainSec * 1000).toLocaleTimeString()}
                            </Card.Text> : null}

                        {jobInfo.isActive ? <Card.Text>
                            <abbr title="Print / Total">Time</abbr>: {jobInfo.printTime} / {jobInfo.totalTime}
                        </Card.Text> : null}
                    </> : null}

                    <Card.Text>Job Owner: <Badge bg="dark">N/A</Badge></Card.Text>

                    {jobInfo?.isActive && jobInfo.jobWillPause ? <>
                        <Card.Subtitle>
                            Job will be paused
                            {jobInfo.pauseRemainSec ?
                                ` after ${secondsToDurationString(jobInfo.pauseRemainSec)}` : null
                            }, please register.
                        </Card.Subtitle>

                        <div className="d-grid gap-2 mt-3">
                            <Button variant="success">Register Job</Button>
                        </div>
                    </> : null}
                </>
            })()}
        </Card.Body>
    </Card>
}

export default PrinterCard
