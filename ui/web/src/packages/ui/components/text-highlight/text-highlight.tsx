type TextHighlightProps = {
  text: string;
  search?: string;
};

export function TextHighlight({ text, search }: TextHighlightProps) {
  if (!search || !text) {
    return <>{text}</>;
  }
  const lowerText = text.toLowerCase();
  const lowerSearch = search.toLowerCase();
  const startIndex = lowerText.indexOf(lowerSearch);

  if (startIndex === -1) {
    return <>{text}</>;
  }

  const endIndex = startIndex + search.length;

  return (
    <>
      {text.substring(0, startIndex)}
      <mark>{text.substring(startIndex, endIndex)}</mark>
      {text.substring(endIndex)}
    </>
  );
}
