import {Fragment, useContext, useMemo} from "react";

import Badge from "react-bootstrap/Badge"
import Container from "react-bootstrap/Container";
import Col from "react-bootstrap/Col"
import Row from "react-bootstrap/Row"

import PrinterCard from "./components/PrinterCard";
import {convertPrinter} from "../types";
import {MoonrakerPrinterState} from "../api";
import {usePrintersQuery} from "./hooks/usePrintersQuery";
import {PrintersAPIUrlBase} from "./printersAPIContext";
import {getPrinterStateInfo, getPrinterStateKeyByValue} from "../utils";

function Home() {
    const apiUrlBase = useContext(PrintersAPIUrlBase);

    // Queries
    const {data: queryData, dataUpdatedAt} = usePrintersQuery();

    const printers = useMemo(() => {
        if (!queryData?.data) return [];

        const data = queryData.data.map(convertPrinter);

        data.sort((a, b) => {
            const aDisconnected = a.state === MoonrakerPrinterState.Disconnected;
            const bDisconnected = b.state === MoonrakerPrinterState.Disconnected;

            if (aDisconnected && !bDisconnected) return 1;
            else if (!aDisconnected && bDisconnected) return -1;
            return a.key!.localeCompare(b.key!);
        })

        return data;
    }, [queryData?.data]);

    const printerStat = useMemo(() => {
        return printers.reduce<Map<MoonrakerPrinterState, number>>((prev, curr) => {
            const {state} = curr;
            const count = (prev.get(state) ?? 0) + 1;
            prev.set(state, count);

            return prev;
        }, new Map<MoonrakerPrinterState, number>());
    }, [printers]);

    return (
        <Container className="py-3" fluid="md">
            <Row xs={1} md={2} className="mb-4">
                <Col className="align-content-center"><h1 className="mb-0">3D Printer Controller</h1></Col>
                <Col className="align-content-center text-end">
                    <p className="fs-6 mb-0">{Array.from(printerStat.entries())
                        .map(([state, count], idx) => {
                            const {stateColor} = getPrinterStateInfo(state);

                            return <Fragment key={state}>
                                {idx !== 0 ? " " : null}
                                <Badge bg={stateColor}>{count}{" "}{getPrinterStateKeyByValue(state)}</Badge>
                            </Fragment>
                        })}
                    </p>
                    <p className="text-muted fw-medium">
                        Last updated at {new Date(dataUpdatedAt).toLocaleTimeString()}
                    </p>
                </Col>
            </Row>

            <Row xs={1} md={2} lg={3} className="g-4">
                {printers
                    .map((printer) => {
                        return <Col key={printer.key}>
                            <PrinterCard printer={printer} apiURLBase={apiUrlBase!}/>
                        </Col>
                    })
                }

                {/*<Col>*/}
                {/*    <Card border="danger">*/}
                {/*        <Card.Header>V400 #2 - <span className="text-danger fw-semibold">Error</span></Card.Header>*/}
                {/*        <Card.Body>*/}
                {/*            <Card.Title>Current Job: FSR_20.gcode</Card.Title>*/}
                {/*            <Card.Subtitle className="mb-2 text-muted">*/}
                {/*                Estimate: 8:22:00, ETA: 6:20 PM*/}
                {/*            </Card.Subtitle>*/}
                {/*            <Card.Text>Time: 01:00:00 / 02:00:00</Card.Text>*/}
                {/*            <Card.Text>Job Id: <Badge>000034</Badge> Job owner: John Doe</Card.Text>*/}
                {/*        </Card.Body>*/}
                {/*    </Card>*/}
                {/*</Col>*/}

                {/*<Col>*/}
                {/*    <Card bg="warning">*/}
                {/*        <Card.Header>V400 #3 - <span className="text-success fw-semibold">Printing</span></Card.Header>*/}
                {/*        <Card.Body>*/}
                {/*            <Card.Title>Current Job: FSR_20.gcode</Card.Title>*/}
                {/*            <Card.Subtitle className="mb-2 text-muted">*/}
                {/*                Estimate: 8:22:00, ETA: 6:20 PM*/}
                {/*            </Card.Subtitle>*/}
                {/*            <Card.Text>Time: 01:00:00 / 02:00:00</Card.Text>*/}
                {/*            <Card.Text>Job Id: <Badge>00000A</Badge> Job owner: John Doe</Card.Text>*/}

                {/*            <Card.Title>Job will be paused after 4m30s</Card.Title>*/}
                {/*            <div className="d-grid gap-2 mt-3">*/}
                {/*                <Button variant="success">Register Job</Button>*/}
                {/*            </div>*/}
                {/*        </Card.Body>*/}
                {/*    </Card>*/}
                {/*</Col>*/}

                {/*<Col>*/}
                {/*    <Card border="success">*/}
                {/*        <Card.Header>V400 #4 - <span className="text-success fw-semibold">Printing</span></Card.Header>*/}
                {/*        <Card.Body>*/}
                {/*            <Card.Title>Current Job: FSR_20.gcode</Card.Title>*/}
                {/*            <Card.Subtitle className="mb-2 text-muted">*/}
                {/*                Estimate: 8:22:00, ETA: 6:20 PM*/}
                {/*            </Card.Subtitle>*/}
                {/*            <Card.Text>Time: 01:00:00 / 02:00:00</Card.Text>*/}
                {/*            <Card.Text>Job Id: <Badge>000034</Badge> Job owner: John Doe</Card.Text>*/}
                {/*        </Card.Body>*/}
                {/*    </Card>*/}
                {/*</Col>*/}

                {/*<Col>*/}
                {/*    <Card bg="danger" text="light">*/}
                {/*        <Card.Header>V400 #5 - <span className="text-light fw-semibold">Idle</span></Card.Header>*/}
                {/*        <Card.Body>*/}
                {/*            <Card.Title>This printer is not open for use</Card.Title>*/}
                {/*            <Card.Subtitle>Please contact admin</Card.Subtitle>*/}

                {/*            <div className="d-grid gap-2 mt-3">*/}
                {/*                <Button>Admin Unlock</Button>*/}
                {/*            </div>*/}
                {/*        </Card.Body>*/}
                {/*    </Card>*/}
                {/*</Col>*/}
            </Row>
        </Container>
    )
        ;
}

export default Home