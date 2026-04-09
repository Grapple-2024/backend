'use client';



import EmptyLayout from '@/layouts/EmptyLayout';



function RootLayout({
  children,
}: Readonly<{
  children: React.ReactNode;
}>) {
  return (
    <EmptyLayout>
      {children}
    </EmptyLayout>
  );
}

export default RootLayout;
