#include <stdio.h>

int array[]={2,3,7,5,2,9,6,9,3,2};

void ShakerSort(int *a,int len)
{
    int T,j,min,max;
	int i = 0;
	int k = len;
	while (i < k) 
    {
        min=i;
        max=i;
        for (j=i+1;j< k;j++) 
        {
		    if (a[j] < a[min]) min = j;
            if (a[j] > a[max]) max = j;
	    }
        T = a[min];
        a[min] = a[i];
	    a[i] = T;

	    if (max == i) 
        {
	        T = a[min];
	        a[min] = a[k];
	 	    a[k] = T;
	    } 
        else 
        {
	        T = a[max];
	        a[max] = a[k];
	        a[k] = T;
	    }
	  
	    i++;
	    k--;
    }
}

int main()
{
    ShakerSort(a,10);
    for (int i=0;i<10;i++)
    {
        cout << "a[" << i << "]= " << a[i] << endl;
    }
    return 0;
}
