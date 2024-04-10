'use client'

import {Inter} from "next/font/google";
import 'bootstrap/dist/css/bootstrap.min.css';
import './global.css'
import {QueryClient, QueryClientProvider} from "@tanstack/react-query";
import {Configuration, PrintersApi} from "@/api";
import {PrintersAPIContext, PrintersAPIUrlBase} from "@/app/printersAPIContext";

const inter = Inter({subsets: ["latin"]});

const queryClient = new QueryClient();

const basePath: string = process.env.NODE_ENV == "development" ?
    'http://localhost:8080/api/v1' : '/api/v1';
const printersAPI = new PrintersApi(new Configuration({basePath}), basePath);

export default function RootLayout({children}: Readonly<{ children: React.ReactNode }>) {
    return (
        <QueryClientProvider client={queryClient}>
            <PrintersAPIContext.Provider value={printersAPI}>
                <PrintersAPIUrlBase.Provider value={basePath}>
                    <html lang="en">
                    <body className={inter.className}>{children}</body>
                    </html>
                </PrintersAPIUrlBase.Provider>
            </PrintersAPIContext.Provider>
        </QueryClientProvider>
    );
}
