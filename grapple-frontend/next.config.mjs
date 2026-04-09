/** @type {import('next').NextConfig} */
const nextConfig = {
  images: {
    remotePatterns: [
      // S3 bucket for user-uploaded assets (avatars, gym logos, banners)
      {
        protocol: 'https',
        hostname: '**.amazonaws.com',
        port: '',
        pathname: '/**',
      },
      // Optional override for a specific image host via env var
      ...(process.env.NEXT_PUBLIC_IMAGE_HOST
        ? [
            {
              protocol: 'https',
              hostname: process.env.NEXT_PUBLIC_IMAGE_HOST,
              port: '',
              pathname: '/**',
            },
          ]
        : []),
    ],
  },
};

export default nextConfig;
