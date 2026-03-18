const ONE_HOUR = 1000 * 60 * 60;

const downloadFile = ({
  content,
  format,
  filename,
}: {
  content: string;
  format: string;
  filename: string;
}) => {
  const encodedUri: string = encodeURI(content);
  const link: HTMLAnchorElement = document.createElement('a');
  link.setAttribute('href', encodedUri);
  const d: Date = new Date();
  const timezoneOffset: number = d.getTimezoneOffset();
  const timezone: number = timezoneOffset / -60;
  const timestamp: number = d.getTime();
  const timezoneTimestamp: number = timestamp + timezone * ONE_HOUR;
  const isoString: string = new Date(timezoneTimestamp).toISOString();
  const [date, time] = isoString.split('T');
  const [timeString] = time.split(':').join('-').split('.');
  const filenameWithExtension: string = `${filename}-${date}-${timeString}.${format}`;
  link.setAttribute('download', filenameWithExtension);
  document.body.append(link); // Required for FF
  link.click(); // This will download the data file.
  link.remove();
};

export const downloadText = ({
  content,
  format,
  filename,
}: {
  content: string;
  format: string;
  filename: string;
}) => {
  const textContent: string = `data:text/${format};charset=utf-8,${content}`;
  downloadFile({ content: textContent, format, filename });
};

export const downloadImage = ({
  content,
  format,
  filename,
}: {
  content: string;
  format: 'jpg' | 'png' | 'ico' | 'gif';
  filename: string;
}) => {
  downloadFile({ content, format, filename });
};

export const download = ({
  content,
  format,
  filename,
}: {
  content: string;
  format: 'jpg' | 'png' | 'ico' | 'gif';
  filename: string;
}) => {
  return {
    image: () => downloadImage({ content, format, filename }),
    text: () => downloadText({ content, format, filename }),
  };
};
