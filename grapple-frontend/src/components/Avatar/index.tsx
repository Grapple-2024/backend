import { Image } from "react-bootstrap"


const Avatar = ({ src, height }: { src: string, height: any }) => {
  return (
    <>
      <Image 
        src={src} 
        style={{ 
          objectFit: 'cover',
          height: height,
          clipPath: 'circle()',
        }} 
        alt="User Avatar Image"/>
    </>
  )
}

export default Avatar;