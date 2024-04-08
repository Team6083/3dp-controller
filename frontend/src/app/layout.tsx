'use client'

import {Inter} from "next/font/google";
import 'bootstrap/dist/css/bootstrap.min.css';
import './global.css'
import {QueryClient, QueryClientProvider} from "@tanstack/react-query";
import {OpenAPI} from "@/api";

const inter = Inter({subsets: ["latin"]});

const queryClient = new QueryClient();
OpenAPI.BASE = 'http://localhost:8080/api/v1';

export default function RootLayout({children}: Readonly<{ children: React.ReactNode }>) {
    return (
        <QueryClientProvider client={queryClient}>
            <html lang="en">
            <body className={inter.className}>{children}</body>
            </html>
        </QueryClientProvider>
    );
}
