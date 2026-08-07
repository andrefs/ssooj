FROM public.ecr.aws/lambda/provided:al2023
RUN dnf install -y poppler-utils && dnf clean all
COPY bootstrap /bootstrap
CMD ["/bootstrap"]
