

# Project 1 **Implement an erasure code into MinIO system**

## 问题分析

Project 1要求我们实现非Reed-solomon纠删码，并且把它融入到MinIO中，能够成功的运行。 我选择实现的纠删码是3+2， 三个数据盘加上两个容错盘的RAID6纠删码。与RAID 5相比，RAID 6增加第二个独立的奇偶校验信息块。两个独立的奇偶系统使用不同的算法，数据的可靠性非常高，任意两块磁盘同时失效时不会影响数据完整性。

在具体实现中，我发现MinIO golang代码其实把Erasure Coding的底层逻辑计算部分给抽离出来了，形成了一个单独的package（https://github.com/klauspost/reedsolomo）， 所以，我们只需要把RAID6算法封装成一个类似的包，实现对应的encode, verify, reconstruct等接口即可。

## 纠删码原理

![image-20220621211831864](C:\Users\YutLan\AppData\Roaming\Typora\typora-user-images\image-20220621211831864.png)

RAID6纠删码的原理如图所示，对于一个数据矩阵，用它左乘一个编码矩阵，得到编码后的结果。 在解码时，我们去掉编码矩阵中丢失数据对应的行。然后左乘编码矩阵的逆矩阵即可。

## 有限伽罗华域

我们首先需要实现的是有限伽罗华域

 **伽罗华域：**

​	我们的编码使用的是伽罗华域，在伽罗华域上的四则运算方式实际上就是是多项式计算。考虑一个byte的bit数是8，我们选择GF(8)来保证每个byte的数据在伽罗华域上都是独特的。

**本原多项式：**

伽罗华域其实本质上是一个群域，伽罗华域的元素可以通过该域上的本原多项式生成。通过本原多项式得到的域，其加法单位元都是 0，乘法单位元是1。以 $\mathrm{GF}\left(2^{3}\right)$ 为例, 指数小于 3 的多项式共 8 个: $0,1, x, x+1, x^{2}, x^{2}+1, x^{2}+x, x^{2}+x+1$ 。其系 数刚好就是 $000,001,010,011,100,101,110,111$, 是 0 到 7 这 8 个数的二进制形式。 在本项目中，我使用的本原多项式是’  Poly84320_g2‘。

**有限伽罗华域中的四则运算：**

**加减：** 有限伽罗华域的加法和剑减法都是使用xor。

```go
func (_ *GF) Add(x, y byte) byte { return x ^ y }
```

**乘法：** 乘法则是取对数做加法，然后取指数。

```
func (gf *GF) Mul(x, y byte) byte {
	if x == 0 || y == 0 {
		return 0
	}
	return gf.exp[uint(gf.log[x])+uint(gf.log[y])]
}
func (gf *GF) Div(x, y byte) byte {
	if x == 0 || y == 0 {
		if y == 0 {
			panic(ErrDivByZero)
		}
		return 0
	}
	return gf.exp[gf.m+uint(gf.log[x])-uint(gf.log[y])]
}
```

伽罗华域的实现在gf.go中，你可以利用如下脚本来演算正确性。

```shell
go test gf_test.go gf.go
```

## 伽罗华域下的矩阵运算

正如前文提到的纠删码encode和decode的过程，都涉及了到了伽罗华域下的矩阵计算，需要对之进行实现，这里参考了（https://github.com/klauspost/reedsolomo）的实现，包含：

**利用高斯消元法进行矩阵求逆**

```go
func (gf *GF) MatrixInvert(m matrix) (matrix, error) {
	if !m.IsSquare() {
		return nil, errNotSquare
	}

	size := len(m)
	work, _ := identityMatrix(size)
	work, _ = m.Augment(work)
	err := gf.gaussianElimination(work)
	if err != nil {
		return nil, err
	}
	return work.SubMatrix(0, size, size, size*2)
}

```

**生成RAID6编码矩阵**

```
func (gf *GF) Raid6EncoderMatrix(rows, cols int) (matrix, error) {
	m, err := newMatrix(rows, cols)
	if err != nil {
		return nil, err
	}
	for c := 0; c < cols; c++ {
		m[c][c] ^= 1
	}
	// set the p row
	for c := 0; c < cols; c++ {
		m[rows-2][c] ^= 1
	}
	// set the q row
	for c := 0; c < cols; c++ {
		m[rows-1][c] ^= gf.Power(byte(c+1), 2)
	}
	return m, nil
}

```

**伽罗华域下的矩阵乘法**

```go
func (gf *GF) MatrixMultiply(m, right matrix) (matrix, error) {
	if len(m[0]) != len(right) {
		return nil, fmt.Errorf("columns on left (%d) is different than rows on right (%d)", len(m[0]), len(right))
	}
	result, _ := newMatrix(len(m), len(right[0]))
	for r, row := range result {
		for c := range row {
			var value byte
			for i := range m[0] {
				value ^= gf.Mul(m[r][i], right[i][c])
			}
			result[r][c] = value
		}
	}
	return result, nil
}
```

## RAID6 Encoder的实现

在完成了伽罗华域和对应的矩阵操作后，我们正式开始实现RAID6 Encoder

```
type Encoder interface {
	Encode(shards [][]byte) error
	Reconstruct(shards [][]byte) error
	ReconstructData(shards [][]byte) error
	// Update(shards [][]byte, newDatashards [][]byte) error
	Split(data []byte) ([][]byte, error)
	// Join(dst io.Writer, shards [][]byte, outSize int) error
}
```

这是minIO里面reedSolomon库可以提供的接口，实际上Join和update是作者为了优化编码框架提供的接口，minIO中并没有使用到。Reconstruct和ReconstructData主要是否要重新申请内存空间，重新放置数据上有所区别，MinIO中只调用到了两个中的一个。因此，后文主要讲一讲Encode和Reconstruct的实现。

```
type Raid6 struct {
	DataShards   int //
	ParityShards int // 2 Number of parity shards, should not be modified.
	Shards       int // Total number of shards. It should be DataShards + 1
	m            matrix
	field        *GF
}
```

我们定义一个Raid6结构体，然后实现这个结构体的对应方法。其中DataShard 是数据盘的数目，ParityShards应该指定为2。

```
func (r *Raid6) Encode(shards [][]byte) error {
	fmt.Println("encoderShards:", shards)
	if len(shards) != r.Shards {
		return ErrShardNoData
	}
	if len(shards[0]) != len(shards[r.DataShards]) {
		return ErrShardSize
	}
	data := shards[0:r.DataShards]
	encoderResult, _ := r.field.MatrixMultiply(r.m, data)
	fmt.Println("encoderResult:", encoderResult)
	copy(shards, encoderResult)

	return nil
}
```

编码过程如上，实现很显然，就是调用了编码矩阵来左乘数据矩阵即可。

```
func (r *Raid6) ReconstructData(shards [][]byte) error {
	if len(shards) != r.Shards {
		return ErrTooFewShards
	}
	subShards := make([][]byte, r.DataShards)
	validIndices := make([]int, r.DataShards)
	invalidIndices := make([]int, 0)
	subMatrixRow := 0
	for matrixRow := 0; matrixRow < r.Shards && subMatrixRow < r.DataShards; matrixRow++ {
		if len(shards[matrixRow]) != 0 {
			subShards[subMatrixRow] = shards[matrixRow]
			validIndices[subMatrixRow] = matrixRow
			subMatrixRow++
		} else {
			invalidIndices = append(invalidIndices, matrixRow)
		}
	}

	subMatrix, _ := newMatrix(r.DataShards, r.DataShards)
	for subMatrixRow, validIndex := range validIndices {
		for c := 0; c < r.DataShards; c++ {
			subMatrix[subMatrixRow][c] = r.m[validIndex][c]
		}
	}
	dataDecodeMatrix, err := r.field.MatrixInvert(subMatrix)
	if err != nil {
		return err
	}
	// reconstructData, err := r.field.MatrixMultiply(dataDecodeMatrix, subShards)
	// copy(shards, reconstructData)
	// fmt.Println("DecoderResult data:", shards)

	reconstructData, err := r.field.MatrixMultiply(dataDecodeMatrix, subShards)
	data := reconstructData[0:r.DataShards]
	encoderResult, _ := r.field.MatrixMultiply(r.m, data)
	copy(shards, encoderResult)
	fmt.Println("DecoderResult:", shards)
	return err
}
```

解码Reconstruct操作则是，先校验数据，看看数据是否存在某一行变为空了（数据丢失），并且记录下丢失的index。如果丢失太多了，则编码矩阵无法求得，就会返回err。否则就会返回解码后的数据。

## 打包框架and导入了minIO

因为在作业提交前不太好就在github上上传库并且把它变成public，本次作业中，我使用的是go mod本地库导入。

在minIO的go.mod文件中添加

```go
replace example.com/galoisfield => /home/lanyuting/minIO/go_prject/erasure_algorithm/lyt_erasure_algorithm/galoisfield
```

然后在使用到纠删码的地方添加

```go
import  （
 "example.com/galoisfield"
）
```

即可。

此外, 把minIO的encoder 从reedsolomon.Encoder修改为galoisfield.Encoder， 即可。

```
type Erasure struct {
	encoder                  func() galoisfield.Encoder
	dataBlocks, parityBlocks int
	blockSize                int64
}
```

此外，因为原本minIO支持的是m+n的容错，默认的测试用例会有10+5的测试，我们需要把这一部分代码注释掉。

## 测试

我们执行go build对minIO进行打包。

```shel
go build
```

然后进行恢复测试

我们创建如下路径，模拟对应的shard。

```
data1/1
data1/2
data1/3
data2/4
data2/5
```

运行如下脚本，进行minio纠删码模式

```
sudo ./minio server /mnt/e/minio/data1{1..3} /mnt/e/minio/data2/{1..2}
```

这样data1就有三个盘，data2有两个盘。

删掉data2会自动恢复，删掉data1就无法恢复。

启动如下

![image-20220622000438452](C:\Users\YutLan\AppData\Roaming\Typora\typora-user-images\image-20220622000438452.png)

无法恢复情况

![image-20220622000510829](C:\Users\YutLan\AppData\Roaming\Typora\typora-user-images\image-20220622000510829.png)
