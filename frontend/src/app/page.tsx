'use client'

import Badge from "react-bootstrap/Badge"
import Button from "react-bootstrap/button"
import Card from "react-bootstrap/Card";
import Container from "react-bootstrap/Container";
import Col from "react-bootstrap/Col"
import Row from "react-bootstrap/Row"
import {useQuery, useQueryClient} from "@tanstack/react-query";
import {moonraker_PrinterState, PrintersService} from "@/api";
import PrinterCard from "@/app/components/PrinterCard";
import {useMemo} from "react";
import {convertPrinter} from "@/types";

function Home() {

    const queryClient = useQueryClient();

    // Queries
    const query = useQuery({
        queryKey: ['printers'],
        queryFn: PrintersService.getPrinters,
        refetchInterval: 5000,
    });

    const printers = useMemo(() => {
        if (!query.data) return [];

        const data = [...query.data]
            .filter((v) => v.state !== moonraker_PrinterState.Disconnected);

        data.sort((a, b) => a.key!.localeCompare(b.key!))

        return data;
    }, [query.data]);

    return (
        <Container className="py-3" fluid="md">
            <h1 className="mb-4">3D Printer Controller</h1>

            <Row xs={1} md={2} lg={3} className="g-4">
                {printers.map((v) => {
                    console.log(v);

                    const printer = convertPrinter(v);

                    return <Col key={v.key}>
                        <PrinterCard printer={printer}/>
                    </Col>
                })}

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