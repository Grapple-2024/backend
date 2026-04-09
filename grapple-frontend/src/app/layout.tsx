'use client';

import './globals.css';
import 'bootstrap/dist/css/bootstrap.min.css';
import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { ReactQueryDevtools } from '@tanstack/react-query-devtools';
import { useState } from 'react';
import { SelectionProvider } from '@/components/Navigation/context';
import { MessagingProvider } from '@/context/message';
import { MobileProvider } from '@/context/mobile';
import { LoadingProvider } from '@/context/loading';
import { EditSeriesProvider } from '@/context/edit-series';
import { UserProvider } from '@/context/user';
import { Providers } from './providers';

export default function RootLayout({ children }: any) {
  const [queryClient] = useState(() => new QueryClient());

  return (
    <html lang="en">
      <head>
        <title>Grapple</title>
        <meta name="description" content="The next generation of martial arts training." />
        <meta name="viewport" content="width=device-width, initial-scale=1.0, viewport-fit=cover, user-scalable=no, maximum-scale=1.0" />
        <link rel="icon" href="/grapple-header.png" sizes="192x192" />
        <link rel="preconnect" href="https://fonts.googleapis.com" {...({ precedence: "default" } as any)} />
        <link rel="preconnect" href="https://fonts.gstatic.com" crossOrigin="anonymous" {...({ precedence: "default" } as any)} />
        <link href="https://fonts.googleapis.com/css2?family=Hind:wght@300;400;500;600;700&family=Lato:ital,wght@0,100;0,300;0,400;0,700;0,900;1,100;1,300;1,400;1,700;1,900&family=Poppins:ital,wght@0,100;0,200;0,300;0,400;0,500;0,600;0,700;0,800;0,900&display=swap" rel="stylesheet" {...({ precedence: "default" } as any)} />
      </head>
      <body>
        <Providers>
          <UserProvider>
            <EditSeriesProvider>
              <LoadingProvider>
                <MobileProvider>
                  <MessagingProvider>
                    <SelectionProvider>
                      <QueryClientProvider client={queryClient}>
                        {children}
                        <ReactQueryDevtools initialIsOpen={false} />
                      </QueryClientProvider>
                    </SelectionProvider>
                  </MessagingProvider>
                </MobileProvider>
              </LoadingProvider>
            </EditSeriesProvider>
          </UserProvider>
        </Providers>
      </body>
    </html>
  );
}
