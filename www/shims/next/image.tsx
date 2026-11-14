import React from "react"

type ImageProps = React.ImgHTMLAttributes<HTMLImageElement> & {
  src: string | { src: string }
  priority?: boolean
  unoptimized?: boolean
  quality?: number
}

const Image = ({ src, priority: _p, unoptimized: _u, quality: _q, ...rest }: ImageProps) => (
  // eslint-disable-next-line @next/next/no-img-element
  <img src={typeof src === "string" ? src : src.src} {...rest} />
)
export default Image
